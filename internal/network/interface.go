// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"time"

	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/netwrap"
	"go.cryptoscope.co/secretstream"
	refs "go.mindeco.de/ssb-refs"
)

// EndpointStat gives some information about a connected peer
type EndpointStat struct {
	ID       *refs.FeedRef
	Addr     net.Addr
	Since    time.Duration
	Endpoint muxrpc.Endpoint
}

type Network interface {
	Connect(ctx context.Context, addr net.Addr) error
	Serve(context.Context, ...muxrpc.HandlerWrapper) error
	GetListenAddr() net.Addr

	GetAllEndpoints() []EndpointStat
	GetEndpointFor(refs.FeedRef) (muxrpc.Endpoint, bool)

	GetConnTracker() ConnTracker

	// websock hack
	HandleHTTP(handler http.Handler)

	io.Closer
}

// ConnTracker decides if connections should be established and keeps track of them
type ConnTracker interface {
	// Active returns true and since when a peer connection is active
	Active(net.Addr) (bool, time.Duration)

	// OnAccept receives a new connection as an argument.
	// If it decides to accept it, it returns true and a context that will be canceled once it should shut down
	// If it decides to deny it, it returns false (and a nil context)
	OnAccept(context.Context, net.Conn) (bool, context.Context)

	// OnClose notifies the tracker that a connection was closed
	OnClose(conn net.Conn) time.Duration

	// Count returns the number of open connections
	Count() uint

	// CloseAll closes all tracked connections
	CloseAll()
}

// GetFeedRefFromAddr uses netwrap to get the secretstream address and then uses ParseFeedRef
func GetFeedRefFromAddr(addr net.Addr) (*refs.FeedRef, error) {
	addr = netwrap.GetAddr(addr, secretstream.NetworkString)
	if addr == nil {
		return nil, errors.New("no shs-bs address found")
	}
	ssAddr := addr.(secretstream.Addr)
	return refs.ParseFeedRef(ssAddr.String())
}
