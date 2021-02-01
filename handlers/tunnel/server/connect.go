package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	refs "go.mindeco.de/ssb-refs"

	"go.cryptoscope.co/muxrpc/v2"
)

type connectArg struct {
	Portal refs.FeedRef `json:"portal"`
	Target refs.FeedRef `json:"target"`
}

type connectWithOriginArg struct {
	connectArg
	Origin refs.FeedRef `json:"origin"` // this should be clear from the shs session already
}

func (rs *roomState) connect(ctx context.Context, req *muxrpc.Request, peerSrc *muxrpc.ByteSource, peerSnk *muxrpc.ByteSink) error {
	// unpack arguments

	var args []connectArg
	err := json.Unmarshal(req.RawArgs, &args)
	if err != nil {
		return fmt.Errorf("connect: invalid arguments: %w", err)
	}

	if n := len(args); n != 1 {
		return fmt.Errorf("connect: expected 1 argument, got %d", n)
	}
	arg := args[0]

	// see if we have and endpoint for the target
	rs.roomsMu.Lock()

	edp, has := rs.rooms["lobby"][arg.Target.Ref()]
	if !has {
		rs.roomsMu.Unlock()
		return fmt.Errorf("no such endpoint")
	}

	// call connect on them
	var argWorigin connectWithOriginArg
	argWorigin.connectArg = arg
	argWorigin.Origin = rs.self

	targetSrc, targetSnk, err := edp.Duplex(ctx, muxrpc.TypeBinary, muxrpc.Method{"tunnel", "connect"}, argWorigin)
	if err != nil {
		delete(rs.rooms["lobby"], arg.Target.Ref())
		rs.updater.Update(rs.rooms["lobby"].asList())
		rs.roomsMu.Unlock()
		return fmt.Errorf("failed to init connect call with target: %w", err)
	}
	rs.roomsMu.Unlock()

	// pipe data
	var cpy muxrpcDuplexCopy
	cpy.ctx, cpy.cancel = context.WithCancel(ctx)

	go cpy.do(targetSnk, peerSrc)
	go cpy.do(peerSnk, targetSrc)

	return nil
}

type muxrpcDuplexCopy struct {
	ctx    context.Context
	cancel context.CancelFunc
}

func (mdc muxrpcDuplexCopy) do(w *muxrpc.ByteSink, r *muxrpc.ByteSource) {
	for r.Next(mdc.ctx) {
		err := r.Reader(func(rd io.Reader) error {
			_, err := io.Copy(w, rd)
			return err
		})
		if err != nil {
			fmt.Println("read failed:", err)
			w.CloseWithError(err)
			mdc.cancel()
			return
		}
	}
	if err := r.Err(); err != nil {
		fmt.Println("src errored:", err)
		w.CloseWithError(err)
		mdc.cancel()
	}

	return
}
