// SPDX-License-Identifier: MIT

package admin

import (
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
)

type dashboardHandler struct {
	r       *render.Renderer
	flashes *weberrors.FlashHelper

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

	pageData := map[string]interface{}{
		"OnlineRefs":  onlineRefs,
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
