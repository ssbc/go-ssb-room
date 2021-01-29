package roomsrv

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/keks/nocomment"
	"go.cryptoscope.co/muxrpc/v2"

	refs "go.mindeco.de/ssb-refs"
	"go.mindeco.de/ssb-rooms/handlers/tunnel"
	"go.mindeco.de/ssb-rooms/handlers/whoami"
	"go.mindeco.de/ssb-rooms/internal/maybemuxrpc"
	"go.mindeco.de/ssb-rooms/internal/network"
)

func (s *Server) initNetwork() error {
	s.authorizer.lst = make(map[string]struct{})

	// TODO: bake this into the authorizer and make it reloadable

	// simple authorized_keys file, new line delimited @feed.xzy
	if f, err := os.Open(s.repo.GetPath("authorized_keys")); err == nil {
		evtAuthedKeys := kitlog.With(s.logger, "event", "authorized_keys")

		// ignore lines starting with #
		rd := nocomment.NewReader(f)
		sc := bufio.NewScanner(rd)
		i := 0
		for sc.Scan() {
			txt := sc.Text()
			if txt == "" {
				continue
			}
			fr, err := refs.ParseFeedRef(txt)
			if err != nil {
				level.Warn(evtAuthedKeys).Log("skipping-line", i+1, "err", err)
				continue
			}
			s.authorizer.Add(*fr)
			i++
		}
		level.Info(evtAuthedKeys).Log("allowing", i)
		f.Close()
	}

	// muxrpc handler creation and authoratization decider
	mkHandler := func(conn net.Conn) (muxrpc.Handler, error) {
		// bypassing badger-close bug to go through with an accept (or not) before closing the bot
		s.closedMu.Lock()
		defer s.closedMu.Unlock()

		// todo: master authorize()er
		remote, err := network.GetFeedRefFromAddr(conn.RemoteAddr())
		if err != nil {
			return nil, fmt.Errorf("sbot: expected an address containing an shs-bs addr: %w", err)
		}
		if s.keyPair.Feed.Equal(remote) {
			return s.master.MakeHandler(conn)
		}

		if s.authorizer.Authorize(conn) {
			return s.public.MakeHandler(conn)
		}

		return nil, fmt.Errorf("not authorized")
	}

	// whoami
	whoami := whoami.New(kitlog.With(s.logger, "unit", "whoami"), s.Whoami())
	s.public.Register(whoami)
	s.master.Register(whoami)

	s.master.Register(manifestPlug)

	// s.master.Register(replicate.NewPlug(s.Users))

	tunnelPlug := tunnel.New(kitlog.With(s.logger, "unit", "tunnel"), s.rootCtx, s.Whoami())
	s.public.Register(tunnelPlug)

	// tcp+shs
	opts := network.Options{
		Logger:              s.logger,
		Dialer:              s.dialer,
		ListenAddr:          s.listenAddr,
		KeyPair:             s.keyPair,
		AppKey:              s.appKey[:],
		MakeHandler:         mkHandler,
		ConnTracker:         s.networkConnTracker,
		BefreCryptoWrappers: s.preSecureWrappers,
		AfterSecureWrappers: s.postSecureWrappers,

		WebsocketAddr: s.wsAddr,
	}

	var err error
	s.Network, err = network.New(opts)
	if err != nil {
		return fmt.Errorf("failed to create network node: %w", err)
	}

	return nil
}

func (srv *Server) Allow(r refs.FeedRef, yes bool) {
	if yes {
		srv.authorizer.Add(r)
	} else {
		srv.authorizer.Remove(r)
	}
}

type listAuthorizer struct {
	mu  sync.Mutex
	lst map[string]struct{}
}

var _ maybemuxrpc.Authorizer = (*listAuthorizer)(nil)

func (a *listAuthorizer) Add(feed refs.FeedRef) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.lst[feed.Ref()] = struct{}{}
}

func (a *listAuthorizer) Remove(feed refs.FeedRef) {
	a.mu.Lock()
	defer a.mu.Unlock()
	delete(a.lst, feed.Ref())
}

func (a *listAuthorizer) Authorize(remote net.Conn) bool {
	remoteID, err := network.GetFeedRefFromAddr(remote.RemoteAddr())
	if err != nil {
		return false
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	_, has := a.lst[remoteID.Ref()]
	return has
}
