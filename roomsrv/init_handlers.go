package roomsrv

import (
	kitlog "github.com/go-kit/kit/log"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	muxrpc "go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/typemux"

	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/alias"
	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/tunnel/server"
	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/whoami"
)

func (s *Server) initHandlers(aliasDB roomdb.AliasService) {

	// inistaniate handler packages
	whoami := whoami.New(s.Whoami())

	tunnelHandler := server.New(
		kitlog.With(s.logger, "unit", "tunnel"),
		s.Whoami(),
		s.StateManager,
	)

	aliasHandler := alias.New(
		kitlog.With(s.logger, "unit", "aliases"),
		s.Whoami(),
		aliasDB,
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
	}
}
