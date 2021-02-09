// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"net"

	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/typemux"

	refs "go.mindeco.de/ssb-refs"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/broadcasts"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/maybemuxrpc"
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

func New(log kitlog.Logger, ctx context.Context, self refs.FeedRef) maybemuxrpc.Plugin {
	mux := typemux.New(log)

	var rs = new(roomState)
	rs.self = self
	rs.logger = log
	rs.updater, rs.broadcaster = broadcasts.NewRoomChanger()
	rs.rooms = make(roomsStateMap)

	go rs.stateTicker(ctx)

	// so far just lobby (v1 rooms)
	rs.rooms["lobby"] = make(roomStateMap)

	mux.RegisterAsync(append(method, "isRoom"), typemux.AsyncFunc(rs.isRoom))
	mux.RegisterAsync(append(method, "ping"), typemux.AsyncFunc(rs.ping))

	mux.RegisterAsync(append(method, "announce"), typemux.AsyncFunc(rs.announce))
	mux.RegisterAsync(append(method, "leave"), typemux.AsyncFunc(rs.leave))

	mux.RegisterSource(append(method, "endpoints"), typemux.SourceFunc(rs.endpoints))

	mux.RegisterDuplex(append(method, "connect"), typemux.DuplexFunc(rs.connect))

	return plugin{
		h: &mux,
	}
}
