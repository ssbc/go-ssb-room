// SPDX-License-Identifier: MIT

package roomsrv

import (
	"context"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"os/user"
	"path/filepath"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.cryptoscope.co/netwrap"

	refs "go.mindeco.de/ssb-refs"
	"go.mindeco.de/ssb-rooms/internal/maybemod/keys"
	"go.mindeco.de/ssb-rooms/internal/maybemod/multicloser"
	"go.mindeco.de/ssb-rooms/internal/maybemuxrpc"
	"go.mindeco.de/ssb-rooms/internal/network"
	"go.mindeco.de/ssb-rooms/internal/repo"
)

type Server struct {
	logger kitlog.Logger

	rootCtx  context.Context
	Shutdown context.CancelFunc
	closers  multicloser.Closer

	closed   bool
	closedMu sync.Mutex
	closeErr error

	Network    network.Network
	appKey     []byte
	listenAddr net.Addr
	wsAddr     string
	dialer     netwrap.Dialer

	loadUnixSock bool

	repo     repo.Interface
	repoPath string
	keyPair  *keys.KeyPair

	networkConnTracker network.ConnTracker
	preSecureWrappers  []netwrap.ConnWrapper
	postSecureWrappers []netwrap.ConnWrapper

	public maybemuxrpc.PluginManager
	master maybemuxrpc.PluginManager

	authorizer listAuthorizer
}

func (s Server) Whoami() refs.FeedRef {
	return s.keyPair.Feed
}

func New(opts ...Option) (*Server, error) {
	var s Server

	s.public = maybemuxrpc.NewPluginManager()
	s.master = maybemuxrpc.NewPluginManager()

	for i, opt := range opts {
		err := opt(&s)
		if err != nil {
			return nil, fmt.Errorf("error applying option #%d: %w", i, err)
		}
	}

	if s.repoPath == "" {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("error getting info on current user: %w", err)
		}

		s.repoPath = filepath.Join(u.HomeDir, ".ssb-go")
	}

	if s.appKey == nil {
		ak, err := base64.StdEncoding.DecodeString("1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
		if err != nil {
			return nil, fmt.Errorf("failed to decode default appkey: %w", err)
		}
		s.appKey = ak
	}

	if s.dialer == nil {
		s.dialer = netwrap.Dial
	}

	if s.listenAddr == nil {
		s.listenAddr = &net.TCPAddr{Port: network.DefaultPort}
	}

	if s.logger == nil {
		logger := kitlog.NewLogfmtLogger(kitlog.NewSyncWriter(os.Stdout))
		logger = kitlog.With(logger, "ts", kitlog.DefaultTimestampUTC, "caller", kitlog.DefaultCaller)
		s.logger = logger
	}

	if s.rootCtx == nil {
		s.rootCtx, s.Shutdown = context.WithCancel(context.Background())
	}

	s.repo = repo.New(s.repoPath)

	if s.keyPair == nil {
		var err error
		s.keyPair, err = repo.DefaultKeyPair(s.repo)
		if err != nil {
			return nil, fmt.Errorf("sbot: failed to get keypair: %w", err)
		}
	}

	if err := s.initNetwork(); err != nil {
		return nil, err
	}

	if s.loadUnixSock {
		if err := s.initUnixSock(); err != nil {
			return nil, err
		}
	}

	return &s, nil
}

// Close closes the bot by stopping network connections and closing the internal databases
func (s *Server) Close() error {
	s.closedMu.Lock()
	defer s.closedMu.Unlock()

	if s.closed {
		return s.closeErr
	}

	closeEvt := kitlog.With(s.logger, "event", "tunserv closing")
	s.closed = true

	if s.Network != nil {
		if err := s.Network.Close(); err != nil {
			s.closeErr = fmt.Errorf("sbot: failed to close own network node: %w", err)
			return s.closeErr
		}
		s.Network.GetConnTracker().CloseAll()
		level.Debug(closeEvt).Log("msg", "connections closed")
	}

	if err := s.closers.Close(); err != nil {
		s.closeErr = err
		return s.closeErr
	}

	level.Info(closeEvt).Log("msg", "closers closed")
	return nil
}
