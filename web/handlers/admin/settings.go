// SPDX-License-Identifier: MIT

package admin

import (
	// "errors"
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

type settingsHandler struct {
	r     *render.Renderer
	urlTo web.URLMaker

	db roomdb.RoomConfig
}

func (h settingsHandler) overview(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	privacyModes := []roomdb.PrivacyMode{roomdb.ModeOpen, roomdb.ModeCommunity, roomdb.ModeRestricted}
	currentMode, err := h.db.GetPrivacyMode(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve current privacy mode: %w", err)
	}

	return map[string]interface{}{
		"CurrentMode":    currentMode,
		"PrivacyModes":   privacyModes,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h settingsHandler) setPrivacy(w http.ResponseWriter, req *http.Request) {
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

	// get the member behind the POST
	currentMember := members.FromContext(req.Context())
	if currentMember == nil {
		err := weberrors.ErrForbidden{Details: fmt.Errorf("not a registered member")}
		h.r.Error(w, req, http.StatusInternalServerError, err)
		return
	}

	// make sure the member is an admin
	if currentMember.Role != roomdb.RoleAdmin {
		err := weberrors.ErrForbidden{Details: fmt.Errorf("yr not an admin! naughty naughty")}
		h.r.Error(w, req, http.StatusInternalServerError, err)
		return
	}

	pmValue := req.Form.Get("privacy_mode")
	pm := roomdb.ParsePrivacyMode(pmValue)
	if pm == roomdb.ModeUnknown {
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("unknown privacy mode was being set: %v", pmValue))
	}

	err := h.db.SetPrivacyMode(req.Context(), pm)
	if err != nil {
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("something went wrong when setting the privacy mode: %w", err))
	}

	// we successfully set the privacy mode! time to redirect to the updated settings overview
	overview := h.urlTo(router.AdminSettings).String()
	http.Redirect(w, req, overview, http.StatusFound)
}
