// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package roomsrv

import (
	muxrpc "github.com/ssbc/go-muxrpc/v2"
	"github.com/ssbc/go-muxrpc/v2/typemux"
	kitlog "go.mindeco.de/log"

	"github.com/ssbc/go-ssb-room/v2/muxrpc/handlers/alias"
	"github.com/ssbc/go-ssb-room/v2/muxrpc/handlers/gossip"
	"github.com/ssbc/go-ssb-room/v2/muxrpc/handlers/signinwithssb"
	"github.com/ssbc/go-ssb-room/v2/muxrpc/handlers/tunnel/server"
	"github.com/ssbc/go-ssb-room/v2/muxrpc/handlers/whoami"
)

// instantiate and register the muxrpc handlers
func (s *Server) initHandlers() {
	// inistaniate handler packages
	whoami := whoami.New(s.Whoami())

	tunnelHandler := server.New(
		kitlog.With(s.logger, "unit", "tunnel"),
		s.netInfo,
		s.StateManager,
		s.Members,
		s.Config,
	)

	aliasHandler := alias.New(
		kitlog.With(s.logger, "unit", "aliases"),
		s.Whoami(),
		s.Aliases,
		s.netInfo,
	)

	siwssbHandler := signinwithssb.New(
		kitlog.With(s.logger, "unit", "auth-with-ssb"),
		s.Whoami(),
		s.netInfo.Domain,
		s.Members,
		s.authWithSSB,
		s.authWithSSBBridge,
	)

	// register muxrpc commands
	registries := []typemux.HandlerMux{s.public, s.master}

	for _, mux := range registries {
		mux.RegisterAsync(muxrpc.Method{"manifest"}, manifest)
		mux.RegisterAsync(muxrpc.Method{"whoami"}, whoami)

		// register old room v1 commands
		tunnelHandler.RegisterTunnel(mux)

		// register new room v2 commands
		tunnelHandler.RegisterRoom(mux)

		var method = muxrpc.Method{"room"}
		mux.RegisterAsync(append(method, "registerAlias"), typemux.AsyncFunc(aliasHandler.Register))
		mux.RegisterAsync(append(method, "revokeAlias"), typemux.AsyncFunc(aliasHandler.Revoke))
		mux.RegisterAsync(append(method, "listAliases"), typemux.AsyncFunc(aliasHandler.List))

		method = muxrpc.Method{"httpAuth"}
		mux.RegisterAsync(append(method, "invalidateAllSolutions"), typemux.AsyncFunc(siwssbHandler.InvalidateAllSolutions))
		mux.RegisterAsync(append(method, "sendSolution"), typemux.AsyncFunc(siwssbHandler.SendSolution))

		method = muxrpc.Method{"gossip"}
		mux.RegisterDuplex(append(method, "ping"), typemux.DuplexFunc(gossip.Ping))
	}
}
