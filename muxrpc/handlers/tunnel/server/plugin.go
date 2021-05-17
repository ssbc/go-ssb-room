// SPDX-License-Identifier: MIT

package server

import (
	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/typemux"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
)

/* manifest:
{
	"announce": "sync",
	"leave": "sync",
	"connect": "duplex",
	"endpoints": "source",
	"isRoom": "async",
	"ping": "sync",
}
*/

func New(log kitlog.Logger, netInfo network.ServerEndpointDetails, m *roomstate.Manager, members roomdb.MembersService, config roomdb.RoomConfig) *Handler {
	var h = new(Handler)
	h.netInfo = netInfo
	h.logger = log
	h.state = m
	h.members = members
	h.config = config

	return h
}

func (h *Handler) Register(mux typemux.HandlerMux, namespace muxrpc.Method) {
	mux.RegisterAsync(append(namespace, "isRoom"), typemux.AsyncFunc(h.metadata))
	mux.RegisterAsync(append(namespace, "ping"), typemux.AsyncFunc(h.ping))

	mux.RegisterAsync(append(namespace, "announce"), typemux.AsyncFunc(h.announce))
	mux.RegisterAsync(append(namespace, "leave"), typemux.AsyncFunc(h.leave))

	mux.RegisterSource(append(namespace, "endpoints"), typemux.SourceFunc(h.endpoints))

	mux.RegisterDuplex(append(namespace, "connect"), connectHandler{
		logger: h.logger,
		self:   h.netInfo.RoomID,
		state:  h.state,
	})
}
