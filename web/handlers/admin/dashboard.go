// SPDX-License-Identifier: MIT

package admin

import (
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
)

type dashboardHandler struct {
	r         *render.Renderer
	roomState *roomstate.Manager
	dbs       Databases
}

func (h dashboardHandler) overview(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	onlineRefs := h.roomState.List()
	onlineCount := len(onlineRefs)
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

	return map[string]interface{}{
		"OnlineRefs":  onlineRefs,
		"OnlineCount": onlineCount,
		"MemberCount": memberCount,
		"InviteCount": inviteCount,
		"DeniedCount": deniedCount,
	}, nil
}
