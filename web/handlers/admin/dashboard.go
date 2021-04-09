// SPDX-License-Identifier: MIT

package admin

import (
	// "errors"
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	// weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
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
	privacyModes := []roomdb.PrivacyMode{roomdb.ModeOpen, roomdb.ModeCommunity, roomdb.ModeRestricted}
	currentMode, err := h.dbs.Config.GetPrivacyMode(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve current privacy mode: %w", err)
	}

	return map[string]interface{}{
		"OnlineRefs":     onlineRefs,
		"OnlineCount":    onlineCount,
		"MemberCount":    memberCount,
		"InviteCount":    inviteCount,
		"DeniedCount":    deniedCount,
		"CurrentMode":    currentMode,
		"PrivacyModes":   privacyModes,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h dashboardHandler) setPrivacy(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	pmValue := req.Form.Get("privacy_mode")
	fmt.Println(pmValue)
	return nil, nil
}
