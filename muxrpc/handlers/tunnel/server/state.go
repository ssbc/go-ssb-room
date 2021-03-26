// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"encoding/json"
	"fmt"
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

	return false, nil
}

func (h *Handler) leave(_ context.Context, req *muxrpc.Request) (interface{}, error) {
	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	h.state.Remove(*ref)

	return false, nil
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

	// cblgh:
	// * reject if key in deny list / DeniedKeysService
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

	has := h.state.AlreadyAdded(*ref, req.Endpoint())
	if !has {
		// just send the current state to the new peer
		toPeer.Update(h.state.List())
	}
	return nil
}

type updateForwarder struct {
	snk *muxrpc.ByteSink
	enc *json.Encoder
}

func newForwarder(snk *muxrpc.ByteSink) updateForwarder {
	enc := json.NewEncoder(snk)
	snk.SetEncoding(muxrpc.TypeJSON)
	return updateForwarder{
		snk: snk,
		enc: enc,
	}
}

func (uf updateForwarder) Update(members []string) error {
	return uf.enc.Encode(members)
}

func (uf updateForwarder) Close() error {
	return uf.snk.Close()
}
