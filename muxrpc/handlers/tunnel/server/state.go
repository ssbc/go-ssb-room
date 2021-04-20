// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	refs "go.mindeco.de/ssb-refs"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.cryptoscope.co/muxrpc/v2"
)

type Handler struct {
	logger kitlog.Logger
	self   refs.FeedRef

	state   *roomstate.Manager
	members roomdb.MembersService
	config  roomdb.RoomConfig
}

func (h *Handler) isRoom(context.Context, *muxrpc.Request) (interface{}, error) {
	level.Debug(h.logger).Log("called", "isRoom")
	return true, nil
}

func (h *Handler) ping(context.Context, *muxrpc.Request) (interface{}, error) {
	now := time.Now().UnixNano() / 1000
	level.Debug(h.logger).Log("called", "ping")
	return now, nil
}

func (h *Handler) announce(_ context.Context, req *muxrpc.Request) (interface{}, error) {
	level.Debug(h.logger).Log("called", "announce")
	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	h.state.AddEndpoint(*ref, req.Endpoint())

	return true, nil
}

func (h *Handler) leave(_ context.Context, req *muxrpc.Request) (interface{}, error) {
	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	h.state.Remove(*ref)

	return true, nil
}

func (h *Handler) endpoints(ctx context.Context, req *muxrpc.Request, snk *muxrpc.ByteSink) error {
	level.Debug(h.logger).Log("called", "endpoints")

	toPeer := newForwarder(snk)

	// for future updates
	h.state.Register(toPeer)

	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return err
	}

	pm, err := h.config.GetPrivacyMode(ctx)
	if err != nil {
		return fmt.Errorf("running with unknown privacy mode")
	}

	switch pm {
	case roomdb.ModeCommunity:
		fallthrough
	case roomdb.ModeRestricted:
		_, err := h.members.GetByFeed(ctx, *ref)
		if err != nil {
			return fmt.Errorf("external user are not allowed to enumerate members")
		}
	}

	h.state.AlreadyAdded(*ref, req.Endpoint())
	toPeer.Update(h.state.List())

	// go func() {
	//	for range time.Tick(3 * time.Second) {
	// 		toPeer.Update(h.state.List())
	// 	}
	// }()
	return nil
}

type updateForwarder struct {
	mu  sync.Mutex // only one caller to forwarder at a time
	snk *muxrpc.ByteSink
	enc *json.Encoder
}

func newForwarder(snk *muxrpc.ByteSink) *updateForwarder {
	enc := json.NewEncoder(snk)
	snk.SetEncoding(muxrpc.TypeJSON)
	return &updateForwarder{
		snk: snk,
		enc: enc,
	}
}

func (uf *updateForwarder) Update(members []string) error {
	uf.mu.Lock()
	defer uf.mu.Unlock()
	return uf.enc.Encode(members)
}

func (uf *updateForwarder) Close() error {
	uf.mu.Lock()
	defer uf.mu.Unlock()
	return uf.snk.Close()
}
