// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"go.cryptoscope.co/muxrpc/v2"
	refs "go.mindeco.de/ssb-refs"
)

// AttendantsUpdate is emitted if a single member joins or leaves.
// Type is either 'joined' or 'left'.
type AttendantsUpdate struct {
	Type string       `json:"type"`
	ID   refs.FeedRef `json:"id"`
}

// AttendantsInitialState is emitted the first time the stream is opened
type AttendantsInitialState struct {
	Type string         `json:"type"`
	IDs  []refs.FeedRef `json:"ids"`
}

func (h *Handler) attendants(ctx context.Context, req *muxrpc.Request, snk *muxrpc.ByteSink) error {

	// get public key from the calling peer
	peer, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return err
	}

	pm, err := h.config.GetPrivacyMode(ctx)
	if err != nil {
		return fmt.Errorf("running with unknown privacy mode")
	}

	if pm == roomdb.ModeCommunity || pm == roomdb.ModeRestricted {
		_, err := h.membersdb.GetByFeed(ctx, *peer)
		if err != nil {
			return fmt.Errorf("external user are not allowed to enumerate members")
		}
	}

	// add peer to the state
	h.state.AddEndpoint(*peer, req.Endpoint())

	// send the current state
	snk.SetEncoding(muxrpc.TypeJSON)
	err = json.NewEncoder(snk).Encode(AttendantsInitialState{
		Type: "state",
		IDs:  h.state.ListAsRefs(),
	})
	if err != nil {
		return err
	}

	// register for future updates
	toPeer := newAttendantsEncoder(snk)
	h.state.RegisterAttendantsUpdates(toPeer)

	return nil
}

// a muxrpc json encoder for endpoints broadcasts
type attendantsJSONEncoder struct {
	mu  sync.Mutex // only one caller to forwarder at a time
	snk *muxrpc.ByteSink
	enc *json.Encoder
}

func newAttendantsEncoder(snk *muxrpc.ByteSink) *attendantsJSONEncoder {
	enc := json.NewEncoder(snk)
	snk.SetEncoding(muxrpc.TypeJSON)
	return &attendantsJSONEncoder{
		snk: snk,
		enc: enc,
	}
}

func (uf *attendantsJSONEncoder) Joined(member refs.FeedRef) error {
	uf.mu.Lock()
	defer uf.mu.Unlock()
	return uf.enc.Encode(AttendantsUpdate{
		Type: "joined",
		ID:   member,
	})
}

func (uf *attendantsJSONEncoder) Left(member refs.FeedRef) error {
	uf.mu.Lock()
	defer uf.mu.Unlock()
	return uf.enc.Encode(AttendantsUpdate{
		Type: "left",
		ID:   member,
	})
}

func (uf *attendantsJSONEncoder) Close() error {
	uf.mu.Lock()
	defer uf.mu.Unlock()
	return uf.snk.Close()
}
