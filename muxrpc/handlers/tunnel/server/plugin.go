// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package server

import (
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/typemux"
	kitlog "go.mindeco.de/log"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomstate"
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

func (h *Handler) RegisterTunnel(mux typemux.HandlerMux) {
	var namespace = muxrpc.Method{"tunnel"}
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

func (h *Handler) RegisterRoom(mux typemux.HandlerMux) {
	var namespace = muxrpc.Method{"room"}
	mux.RegisterAsync(append(namespace, "metadata"), typemux.AsyncFunc(h.metadata))
	mux.RegisterAsync(append(namespace, "ping"), typemux.AsyncFunc(h.ping))

	mux.RegisterSource(append(namespace, "attendants"), typemux.SourceFunc(h.attendants))

	mux.RegisterDuplex(append(namespace, "connect"), connectHandler{
		logger: h.logger,
		self:   h.netInfo.RoomID,
		state:  h.state,
	})
}
