// SPDX-License-Identifier: MIT

package admin

import (
	"fmt"
	"net/http"
	"time"

	"go.mindeco.de/http/render"
	"go.mindeco.de/log/level"
	"go.mindeco.de/logging"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomstate"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
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
		ctx     = req.Context()
		roomRef = h.netInfo.RoomID.Ref()

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
	onlineMembers := make([]roomdb.Member, len(onlineRefs))
	for i, ref := range onlineRefs {
		var err error
		onlineMembers[i], err = h.dbs.Members.GetByFeed(ctx, ref)
		if err != nil {
			// TODO: do we want to show "external users" (non-members) on the dashboard?
			return nil, fmt.Errorf("failed to lookup online member: %w", err)
		}
	}

	memberCount, err := h.dbs.Members.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}

	inviteCount, err := h.dbs.Invites.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count invites: %w", err)
	}

	deniedCount, err := h.dbs.DeniedKeys.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to count denied keys: %w", err)
	}

	pageData := map[string]interface{}{
		"RoomRef":       roomRef,
		"OnlineMembers": onlineMembers,
		"OnlineCount":   onlineCount,
		"MemberCount":   memberCount,
		"InviteCount":   inviteCount,
		"DeniedCount":   deniedCount,
	}

	pageData["Flashes"], err = h.flashes.GetAll(w, req)
	if err != nil {
		return nil, err
	}

	return pageData, nil
}
