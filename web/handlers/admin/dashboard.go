// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	refs "github.com/ssbc/go-ssb-refs"
	"go.mindeco.de/http/render"
	"go.mindeco.de/log/level"
	"go.mindeco.de/logging"

	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/roomstate"
	weberrors "github.com/ssbc/go-ssb-room/v2/web/errors"
)

type dashboardHandler struct {
	r       *render.Renderer
	flashes *weberrors.FlashHelper

	roomState *roomstate.Manager
	netInfo   network.ServerEndpointDetails
	dbs       Databases
}

func (h dashboardHandler) overview(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	var (
		err     error
		ctx     = req.Context()
		roomRef = h.netInfo.RoomID.String()

		onlineRefs   []refs.FeedRef
		refsUpdateCh = make(chan []refs.FeedRef)
		onlineCount  = -1
	)

	// this is an attempt to sidestep the _dashboard doesn't render_ bug (issue #210)
	// first we retreive the member state via a goroutine in the background
	go func() {
		refsUpdateCh <- h.roomState.ListAsRefs()
	}()

	// if it doesn't complete in 10 seconds the slice stays empty and onlineCount remains -1 (to indicate a problem)
	select {
	case <-time.After(10 * time.Second):
		logger := logging.FromContext(ctx)
		level.Warn(logger).Log("event", "didnt retreive room state in time")

	case onlineRefs = <-refsUpdateCh:
		onlineCount = len(onlineRefs)
	}

	// in the timeout case, nothing will happen here since the onlineRefs slice is empty
	onlineUsers := make([]connectedUser, len(onlineRefs))
	for i, ref := range onlineRefs {
		// try to get the member
		onlineUsers[i].Member, err = h.dbs.Members.GetByFeed(ctx, ref)
		if err != nil {
			if !errors.Is(err, roomdb.ErrNotFound) { // any other error can't be handled here
				return nil, fmt.Errorf("failed to lookup online member: %w", err)
			}

			// if there is no member for this ref present it as role unknown
			onlineUsers[i].ID = -1
			onlineUsers[i].PubKey = ref
			onlineUsers[i].Role = roomdb.RoleUnknown
		}
	}

	memberCount, err := h.dbs.Members.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}

	inviteCount, err := h.dbs.Invites.Count(ctx, true)
	if err != nil {
		return nil, fmt.Errorf("failed to count active invites: %w", err)
	}

	deniedCount, err := h.dbs.DeniedKeys.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count denied keys: %w", err)
	}

	pageData := map[string]interface{}{
		"RoomRef":     roomRef,
		"OnlineUsers": onlineUsers,
		"OnlineCount": onlineCount,
		"MemberCount": memberCount,
		"InviteCount": inviteCount,
		"DeniedCount": deniedCount,
	}

	pageData["Flashes"], err = h.flashes.GetAll(w, req)
	if err != nil {
		return nil, err
	}

	return pageData, nil
}

// connectedUser defines how we want to present a connected user
type connectedUser struct {
	roomdb.Member
}

// if the member has an alias, use the first one. Otherwise use the public key
func (dm connectedUser) String() string {
	if len(dm.Aliases) > 0 {
		return dm.Aliases[0].Name
	}
	return dm.PubKey.String()
}
