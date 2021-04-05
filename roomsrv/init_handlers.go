// SPDX-License-Identifier: MIT

package roomsrv

import (
	kitlog "github.com/go-kit/kit/log"
	muxrpc "go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/typemux"

	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/alias"
	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/gossip"
	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/tunnel/server"
	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/whoami"
)

// instantiate and register the muxrpc handlers
func (s *Server) initHandlers() {
	// inistaniate handler packages
	whoami := whoami.New(s.Whoami())

	tunnelHandler := server.New(
		kitlog.With(s.logger, "unit", "tunnel"),
		s.Whoami(),
		s.StateManager,
		s.Members,
		s.Config,
	)

	aliasHandler := alias.New(
		kitlog.With(s.logger, "unit", "aliases"),
		s.Whoami(),
		s.Aliases,
		s.domain,
	)

	siwssbHandler := signinwithssb.New(
		kitlog.With(s.logger, "unit", "auth-with-ssb"),
		s.Whoami(),
		s.domain,
		s.Members,
		s.authWithSSB,
		s.authWithSSBBridge,
	)

	// register muxrpc commands
	registries := []typemux.HandlerMux{s.public, s.master}

	for _, mux := range registries {
		mux.RegisterAsync(muxrpc.Method{"manifest"}, manifest)
		mux.RegisterAsync(muxrpc.Method{"whoami"}, whoami)

		// register tunnel.connect etc twice (as tunnel.* and room.*)
		var method = muxrpc.Method{"tunnel"}
		tunnelHandler.Register(mux, method)

		method = muxrpc.Method{"room"}
		tunnelHandler.Register(mux, method)

		mux.RegisterAsync(append(method, "registerAlias"), typemux.AsyncFunc(aliasHandler.Register))
		mux.RegisterAsync(append(method, "revokeAlias"), typemux.AsyncFunc(aliasHandler.Revoke))

		method = muxrpc.Method{"httpAuth"}
		mux.RegisterAsync(append(method, "invalidateAllSolutions"), typemux.AsyncFunc(siwssbHandler.InvalidateAllSolutions))
		mux.RegisterAsync(append(method, "sendSolution"), typemux.AsyncFunc(siwssbHandler.SendSolution))

		method = muxrpc.Method{"gossip"}
		mux.RegisterDuplex(append(method, "ping"), typemux.DuplexFunc(gossip.Ping))
	}
}
