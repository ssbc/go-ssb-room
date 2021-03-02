// SPDX-License-Identifier: MIT

package server

import (
	"net"

	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/typemux"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemuxrpc"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	refs "go.mindeco.de/ssb-refs"
)

const name = "tunnel"

var method muxrpc.Method = muxrpc.Method{name}

type plugin struct {
	h   muxrpc.Handler
	log kitlog.Logger
}

func (plugin) Name() string              { return name }
func (plugin) Method() muxrpc.Method     { return method }
func (p plugin) Handler() muxrpc.Handler { return p.h }
func (plugin) Authorize(net.Conn) bool   { return true }

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

func New(log kitlog.Logger, self refs.FeedRef, m *roomstate.Manager) maybemuxrpc.Plugin {
	mux := typemux.New(log)

	var h = new(handler)
	h.self = self
	h.logger = log
	h.state = m

	mux.RegisterAsync(append(method, "isRoom"), typemux.AsyncFunc(h.isRoom))
	mux.RegisterAsync(append(method, "ping"), typemux.AsyncFunc(h.ping))

	mux.RegisterAsync(append(method, "announce"), typemux.AsyncFunc(h.announce))
	mux.RegisterAsync(append(method, "leave"), typemux.AsyncFunc(h.leave))

	mux.RegisterSource(append(method, "endpoints"), typemux.SourceFunc(h.endpoints))

	mux.RegisterDuplex(append(method, "connect"), typemux.DuplexFunc(h.connect))

	return plugin{
		h: &mux,
	}
}
