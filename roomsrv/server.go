// SPDX-License-Identifier: MIT

// Package roomsrv implements the muxrpc server for all the room related code.
// It ties the muxrpc/handlers packages and network listeners together.
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
	"go.cryptoscope.co/muxrpc/v2/typemux"
	"go.cryptoscope.co/netwrap"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/multicloser"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	refs "go.mindeco.de/ssb-refs"
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

	domain string // the DNS domain name of the room

	loadUnixSock bool

	repo     repo.Interface
	repoPath string
	keyPair  *keys.KeyPair

	networkConnTracker network.ConnTracker
	preSecureWrappers  []netwrap.ConnWrapper
	postSecureWrappers []netwrap.ConnWrapper

	public typemux.HandlerMux
	master typemux.HandlerMux

	authorizer roomdb.MembersService

	StateManager *roomstate.Manager

	Members    roomdb.MembersService
	DeniedKeys roomdb.DeniedKeysService
	Aliases    roomdb.AliasesService

	authWithSSB       roomdb.AuthWithSSBService
	authWithSSBBridge *signinwithssb.SignalBridge
	Config            roomdb.RoomConfig
}

func (s Server) Whoami() refs.FeedRef {
	return s.keyPair.Feed
}

func New(
	membersdb roomdb.MembersService,
	deniedkeysdb roomdb.DeniedKeysService,
	aliasdb roomdb.AliasesService,
	awsdb roomdb.AuthWithSSBService,
	bridge *signinwithssb.SignalBridge,
	config roomdb.RoomConfig,
	domainName string,
	opts ...Option,
) (*Server, error) {
	var s Server
	s.authorizer = membersdb

	s.Members = membersdb
	s.DeniedKeys = deniedkeysdb
	s.Aliases = aliasdb
	s.Config = config

	s.authWithSSB = awsdb
	s.authWithSSBBridge = bridge

	s.domain = domainName

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

	s.public = typemux.New(kitlog.With(s.logger, "mux", "public"))
	s.master = typemux.New(kitlog.With(s.logger, "mux", "master"))

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

	s.StateManager = roomstate.NewManager(s.rootCtx, s.logger)

	s.initHandlers()

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
