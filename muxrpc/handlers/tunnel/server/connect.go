// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ssbc/go-muxrpc/v2"
	kitlog "go.mindeco.de/log"
	"go.mindeco.de/log/level"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/ssbc/go-ssb-room/v2/roomstate"
)

type ConnectArg struct {
	Portal refs.FeedRef `json:"portal"` // the room server
	Target refs.FeedRef `json:"target"` // which peer the initiator/caller wants to be tunneld to
}

type connectWithOriginArg struct {
	ConnectArg
	Origin refs.FeedRef `json:"origin"` // who started the call
}

type connectHandler struct {
	logger kitlog.Logger
	self   refs.FeedRef

	state *roomstate.Manager
}

// HandleConnect for tunnel.connect makes sure peers whos muxrpc session ends are removed from the room state
func (h connectHandler) HandleConnect(ctx context.Context, edp muxrpc.Endpoint) {
	// block until the channel is closed when the rpc session ends
	<-ctx.Done()

	peer, err := network.GetFeedRefFromAddr(edp.Remote())
	if err != nil {
		return
	}

	h.state.Remove(*peer)
}

// HandleDuplex here implements the tunnel.connect behavior of the server-side. It receives incoming events
func (h connectHandler) HandleDuplex(ctx context.Context, req *muxrpc.Request, peerSrc *muxrpc.ByteSource, peerSnk *muxrpc.ByteSink) error {
	// unpack arguments
	var args []ConnectArg
	err := json.Unmarshal(req.RawArgs, &args)
	if err != nil {
		return fmt.Errorf("connect: invalid arguments: %w", err)
	}

	if n := len(args); n != 1 {
		return fmt.Errorf("connect: expected 1 argument, got %d", n)
	}
	arg := args[0]

	if !arg.Portal.Equal(&h.self) {
		return fmt.Errorf("talking to the wrong room")
	}

	// who made the call
	caller, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return err
	}

	// make sure they dont want to connect to themselves
	if caller.Equal(&arg.Target) {
		return fmt.Errorf("can't connect to self")
	}

	// see if we have and endpoint for the target
	edp, has := h.state.Has(arg.Target)
	if !has {
		return fmt.Errorf("could not connect to:%s", arg.Target.Ref())
	}

	// call connect on them
	var argWorigin connectWithOriginArg
	argWorigin.ConnectArg = arg
	argWorigin.Origin = *caller

	targetSrc, targetSnk, err := edp.Duplex(ctx, muxrpc.TypeBinary, muxrpc.Method{"tunnel", "connect"}, argWorigin)
	if err != nil {
		return fmt.Errorf("could not connect to:%s", arg.Target.Ref())
	}

	// pipe data between caller and target
	var cpy muxrpcDuplexCopy
	cpy.logger = kitlog.With(h.logger, "caller", caller.ShortRef(), "target", arg.Target.ShortRef())
	cpy.ctx, cpy.cancel = context.WithCancel(ctx)

	go cpy.do(targetSnk, peerSrc)
	go cpy.do(peerSnk, targetSrc)

	return nil
}

type muxrpcDuplexCopy struct {
	ctx    context.Context
	cancel context.CancelFunc

	logger kitlog.Logger
}

func (mdc muxrpcDuplexCopy) do(w *muxrpc.ByteSink, r *muxrpc.ByteSource) {
	for r.Next(mdc.ctx) {
		err := r.Reader(func(rd io.Reader) error {
			_, err := io.Copy(w, rd)
			return err
		})
		if err != nil {
			level.Warn(mdc.logger).Log("event", "read failed", "err", err)
			w.CloseWithError(err)
			mdc.cancel()
			return
		}
	}
	if err := r.Err(); err != nil {
		level.Warn(mdc.logger).Log("event", "source errored", "err", err)
		// TODO: remove reading side from state?!
		w.CloseWithError(err)
		mdc.cancel()
	}

	return
}
