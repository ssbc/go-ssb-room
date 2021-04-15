// SPDX-License-Identifier: MIT

package admin

import (
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
)

type dashboardHandler struct {
	r       *render.Renderer
	flashes *weberrors.FlashHelper

	roomState *roomstate.Manager
	netInfo   network.ServerEndpointDetails
	dbs       Databases
}

func (h dashboardHandler) overview(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	roomRef := h.netInfo.RoomID.Ref()

	onlineRefs := h.roomState.List()

	onlineMembers := make([]roomdb.Member, len(onlineRefs))
	for i := range onlineRefs {
		ref, err := refs.ParseFeedRef(onlineRefs[i])
		if err != nil {
			return nil, fmt.Errorf("failed to parse online ref: %w", err)
		}
		onlineMembers[i], err = h.dbs.Members.GetByFeed(req.Context(), *ref)
		if err != nil {
			// TODO: do we want to show "external users" (non-members) on the dashboard?
			return nil, fmt.Errorf("failed to lookup online member: %w", err)
		}
	}

	onlineCount := len(onlineMembers)

	memberCount, err := h.dbs.Members.Count(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to count members: %w", err)
	}

	inviteCount, err := h.dbs.Invites.Count(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to count invites: %w", err)
	}

	deniedCount, err := h.dbs.DeniedKeys.Count(req.Context())
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
