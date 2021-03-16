// SPDX-License-Identifier: MIT

package roomsrv

import (
	"fmt"
	"net"

	"go.cryptoscope.co/muxrpc/v2"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
)

// opens the shs listener for TCP connections
func (s *Server) initNetwork() error {
	// muxrpc handler creation and authoratization decider
	mkHandler := func(conn net.Conn) (muxrpc.Handler, error) {
		s.closedMu.Lock()
		defer s.closedMu.Unlock()

		remote, err := network.GetFeedRefFromAddr(conn.RemoteAddr())
		if err != nil {
			return nil, fmt.Errorf("sbot: expected an address containing an shs-bs addr: %w", err)
		}

		if s.keyPair.Feed.Equal(remote) {
			return &s.master, nil
		}

		if s.authorizer.HasFeed(s.rootCtx, *remote) {
			return &s.public, nil
		}

		return nil, fmt.Errorf("not authorized")
	}

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

// Allow adds (if yes==true) the passed reference to the list of peers that are allowed to connect to the server,
// yes==false removes it.
func (s *Server) Allow(r refs.FeedRef, yes bool) {
	if yes {
		s.authorizer.Add(s.rootCtx, r)
	} else {
		s.authorizer.RemoveFeed(s.rootCtx, r)
	}
}
