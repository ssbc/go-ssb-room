// SPDX-License-Identifier: MIT

package admin

import (
	// "errors"
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
)

type settingsHandler struct {
	r     *render.Renderer
	urlTo web.URLMaker
	db    roomdb.RoomConfig
	loc   *i18n.Helper
}

func (h settingsHandler) overview(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	privacyModes := []roomdb.PrivacyMode{roomdb.ModeOpen, roomdb.ModeCommunity, roomdb.ModeRestricted}

	currentMode, err := h.db.GetPrivacyMode(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve current privacy mode: %w", err)
	}

	currentLanguage, err := h.db.GetDefaultLanguage(req.Context())
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve current privacy mode: %w", err)
	}

	return map[string]interface{}{
		"CurrentMode":     currentMode,
		"CurrentLanguage": h.loc.ChooseTranslation(currentLanguage),
		"PrivacyModes":    privacyModes,
		csrf.TemplateTag:  csrf.TemplateField(req),
	}, nil
}

func (h settingsHandler) setLanguage(w http.ResponseWriter, req *http.Request) {
	if !h.verifyPostRequirements(w, req) {
		return
	}

	// handles error cases & make sures the member is an admin
	currentMember := h.getMember(w, req)
	if currentMember == nil {
		return
	}

	langTag := req.Form.Get("lang")

	err := h.db.SetDefaultLanguage(req.Context(), langTag)
	if err != nil {
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("something went wrong when setting the default language: %w", err))
	}

	// we successfully set the default language! time to redirect to the updated settings overview
	h.redirect(router.AdminSettings, w, req)
}

func (h settingsHandler) setPrivacy(w http.ResponseWriter, req *http.Request) {
	if !h.verifyPostRequirements(w, req) {
		return
	}
	// handles error cases & make sures the member is an admin
	currentMember := h.getMember(w, req)
	if currentMember == nil {
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
	h.redirect(router.AdminSettings, w, req)
}

/* common-use functions */

func (h settingsHandler) getMember(w http.ResponseWriter, req *http.Request) *roomdb.Member {
	// get the member behind the POST
	currentMember := members.FromContext(req.Context())
	if currentMember == nil {
		err := weberrors.ErrForbidden{Details: fmt.Errorf("not a registered member")}
		h.r.Error(w, req, http.StatusInternalServerError, err)
		return nil
	}

	// make sure the member is an admin
	if currentMember.Role != roomdb.RoleAdmin {
		err := weberrors.ErrForbidden{Details: fmt.Errorf("yr not an admin! naughty naughty")}
		h.r.Error(w, req, http.StatusInternalServerError, err)
		return nil
	}
	return currentMember
}

func (h settingsHandler) verifyPostRequirements(w http.ResponseWriter, req *http.Request) bool {
	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return false
	}
	if err := req.ParseForm(); err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return false
	}
	return true
}

func (h settingsHandler) redirect(route string, w http.ResponseWriter, req *http.Request) {
	overview := h.urlTo(route).String()
	http.Redirect(w, req, overview, http.StatusSeeOther)
}
