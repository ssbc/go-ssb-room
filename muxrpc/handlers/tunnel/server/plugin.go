// SPDX-License-Identifier: MIT

package server

import (
	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/typemux"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	refs "go.mindeco.de/ssb-refs"
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

func New(log kitlog.Logger, self refs.FeedRef, m *roomstate.Manager) *Handler {

	var h = new(Handler)
	h.self = self
	h.logger = log
	h.state = m

	return h
}

func (h *Handler) Register(mux typemux.HandlerMux, namespace muxrpc.Method) {
	mux.RegisterAsync(append(namespace, "isRoom"), typemux.AsyncFunc(h.isRoom))
	mux.RegisterAsync(append(namespace, "ping"), typemux.AsyncFunc(h.ping))

	mux.RegisterAsync(append(namespace, "announce"), typemux.AsyncFunc(h.announce))
	mux.RegisterAsync(append(namespace, "leave"), typemux.AsyncFunc(h.leave))

	mux.RegisterSource(append(namespace, "endpoints"), typemux.SourceFunc(h.endpoints))

	mux.RegisterDuplex(append(namespace, "connect"), typemux.DuplexFunc(h.connect))
}
