// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"

	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"
)

type Handler struct {
	logger kitlog.Logger

	netInfo network.ServerEndpointDetails
	state   *roomstate.Manager
	members roomdb.MembersService
	config  roomdb.RoomConfig
}

type MetadataReply struct {
	Name       string   `json:"name"`
	Membership bool     `json:"membership"`
	Features   []string `json:"features"`
}

func (h *Handler) metadata(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	pm, err := h.config.GetPrivacyMode(ctx)
	if err != nil {
		return nil, err
	}

	var reply MetadataReply
	reply.Name = h.netInfo.Domain

	// check if caller is a member
	if _, err := h.members.GetByFeed(ctx, *ref); err != nil {
		if !errors.Is(err, roomdb.ErrNotFound) {
			return nil, err
		}
		// already initialized as false, just to be clear
		reply.Membership = false
	} else {
		reply.Membership = true
	}

	// always-on features
	reply.Features = []string{
		"tunnel",
		"httpAuth",
		"httpInvite",
		// TODO: add "room2" once implemented
	}

	if pm == roomdb.ModeOpen {
		reply.Features = append(reply.Features, "room1")
	}

	if pm == roomdb.ModeOpen || pm == roomdb.ModeCommunity {
		reply.Features = append(reply.Features, "alias")
	}

	return reply, nil
}

func (h *Handler) ping(context.Context, *muxrpc.Request) (interface{}, error) {
	now := time.Now().UnixNano() / 1000
	return now, nil
}

func (h *Handler) announce(_ context.Context, req *muxrpc.Request) (interface{}, error) {
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
	toPeer := newForwarder(snk)

	// for future updates
	h.state.Register(toPeer)

	// get public key from the calling peer
	peer, err := network.GetFeedRefFromAddr(req.RemoteAddr())
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
		_, err := h.members.GetByFeed(ctx, *peer)
		if err != nil {
			return fmt.Errorf("external user are not allowed to enumerate members")
		}
	}

	// add the peer to the room state if they arent already
	h.state.AlreadyAdded(*peer, req.Endpoint())

	// update the peer with
	toPeer.Update(h.state.List())

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
