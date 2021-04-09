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
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
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

func (h dashboardHandler) setPrivacy(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request"))
		return
	}

	if err := req.ParseForm(); err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request: %w", err))
		return
	}

	pmValue := req.Form.Get("privacy_mode")

	pm := roomdb.ParsePrivacyMode(pmValue)
	if pm == roomdb.ModeUnknown {
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("unknown privacy mode was being set: %v", pmValue))
	}

	err := h.dbs.Config.SetPrivacyMode(req.Context(), pm)
	if err != nil {
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("something went wrong when setting the privacy mode: %w", err))
	}

	urlTo := web.NewURLTo(router.CompleteApp())
	dashboard := urlTo(router.AdminDashboard).String()
	http.Redirect(w, req, dashboard, http.StatusFound)
}
