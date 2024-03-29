// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package server

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ssbc/go-muxrpc/v2"
	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
)

type Member struct {
	ID refs.FeedRef `json:"id"`
}

func (h *Handler) members(ctx context.Context, req *muxrpc.Request, snk *muxrpc.ByteSink) error {
	peer, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return err
	}

	pm, err := h.config.GetPrivacyMode(ctx)
	if err != nil {
		return fmt.Errorf("running with unknown privacy mode: %w", err)
	}

	if pm == roomdb.ModeCommunity || pm == roomdb.ModeRestricted {
		_, err := h.membersdb.GetByFeed(ctx, peer)
		if err != nil {
			return fmt.Errorf("external users are not allowed to list members: %w", err)
		}
	}

	members, err := h.membersdb.List(ctx)
	if err != nil {
		return fmt.Errorf("error listing members: %w", err)
	}

	snk.SetEncoding(muxrpc.TypeJSON)

	for _, member := range members {
		if err = json.NewEncoder(snk).Encode([]Member{
			{
				ID: member.PubKey,
			},
		}); err != nil {
			return fmt.Errorf("encoder error: %w", err)
		}
	}

	return snk.Close()
}
