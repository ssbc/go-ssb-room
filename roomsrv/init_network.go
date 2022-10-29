// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package roomsrv

import (
	"fmt"
	"net"

	"github.com/ssbc/go-muxrpc/v2"

	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
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

		pm, err := s.Config.GetPrivacyMode(s.rootCtx)
		if err != nil {
			return nil, fmt.Errorf("running with unknown privacy mode")
		}

		// if privacy mode is restricted, deny connections from non-members
		if pm == roomdb.ModeRestricted {
			if _, err := s.Members.GetByFeed(s.rootCtx, *remote); err != nil {
				return nil, fmt.Errorf("access restricted to members")
			}
		}

		// if feed is in the deny list, deny their connection
		if s.DeniedKeys.HasFeed(s.rootCtx, *remote) {
			return nil, fmt.Errorf("this key has been banned")
		}

		// for community + open modes, allow all connections
		return &s.public, nil
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
	}

	var err error
	s.Network, err = network.New(opts)
	if err != nil {
		return fmt.Errorf("failed to create network node: %w", err)
	}

	return nil
}
