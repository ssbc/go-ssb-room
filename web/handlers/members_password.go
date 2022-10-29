// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gorilla/csrf"
	hibp "github.com/mattevans/pwned-passwords"
	"go.mindeco.de/http/render"

	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web"
	weberrs "github.com/ssbc/go-ssb-room/v2/web/errors"
	"github.com/ssbc/go-ssb-room/v2/web/members"
	"github.com/ssbc/go-ssb-room/v2/web/router"
)

type membersHandler struct {
	r     *render.Renderer
	urlTo web.URLMaker
	fh    *weberrs.FlashHelper

	authFallbackDB roomdb.AuthFallbackService

	leakedLookup func(string) (bool, error)
}

func newMembersHandler(devMode bool, r *render.Renderer, urlTo web.URLMaker, fh *weberrs.FlashHelper, db roomdb.AuthFallbackService) membersHandler {
	mh := membersHandler{
		r:     r,
		urlTo: urlTo,
		fh:    fh,

		authFallbackDB: db,
	}

	// we dont want to need network for our tests.
	if devMode {
		mh.leakedLookup = func(_ string) (bool, error) {
			return false, nil
		}
	} else {
		// Init the have-i-been-pwned client for insecure password checks.
		const storeExpiry = 1 * time.Hour
		hibpClient := hibp.NewClient(storeExpiry)

		mh.leakedLookup = hibpClient.Pwned.Compromised
	}

	return mh
}

func (mh membersHandler) changePasswordForm(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	resetToken := req.URL.Query().Get("token")
	if members.FromContext(req.Context()) == nil && resetToken == "" {
		return nil, weberrs.ErrNotAuthorized
	}

	// you can't do anything with a wrong/guessed token

	var pageData = make(map[string]interface{})
	pageData[csrf.TemplateTag] = csrf.TemplateField(req)

	var err error
	pageData["Flashes"], err = mh.fh.GetAll(w, req)
	if err != nil {
		return nil, err
	}

	pageData["ResetToken"] = resetToken

	return pageData, nil
}

func (mh membersHandler) changePassword(w http.ResponseWriter, req *http.Request) {
	var (
		ctx         = req.Context()
		memberID    = int64(-1)
		redirectURL = req.Header.Get("Referer")

		resetToken string
	)

	if redirectURL == "" {
		http.Error(w, "TODO: add correct redirect handling", http.StatusInternalServerError)
		return
	}

	if req.Method != http.MethodPost {
		mh.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("expected POST method"))
		return
	}

	err := req.ParseForm()
	if err != nil {
		mh.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	resetToken = req.FormValue("reset-token")
	if m := members.FromContext(ctx); m != nil {
		memberID = m.ID

		// shouldn't have both token and logged in user
		if resetToken != "" {
			mh.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("can't have logged in user and reset-token present. Log out and try again"))
			return
		}
	}

	// check the passwords match and it hasnt been pwned
	repeat := req.FormValue("repeat-password")
	newpw := req.FormValue("new-password")

	if newpw != repeat {
		mh.fh.AddError(w, req, weberrs.ErrGenericLocalized{Label: "ErrorPasswordDidntMatch"})
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	if len(newpw) < 10 {
		mh.fh.AddError(w, req, weberrs.ErrGenericLocalized{Label: "ErrorPasswordTooShort"})
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	isPwned, err := mh.leakedLookup(newpw)
	if err != nil {
		mh.r.Error(w, req, http.StatusInternalServerError, fmt.Errorf("have-i-been-pwned client failed: %w", err))
		return
	}

	if isPwned {
		mh.fh.AddError(w, req, weberrs.ErrGenericLocalized{Label: "ErrorPasswordLeaked"})
		http.Redirect(w, req, redirectURL, http.StatusSeeOther)
		return
	}

	// update the password
	if resetToken == "" {
		err = mh.authFallbackDB.SetPassword(ctx, memberID, newpw)
	} else {
		err = mh.authFallbackDB.SetPasswordWithToken(ctx, resetToken, newpw)
	}

	// add flash msg about the outcome and redirect the user
	if err != nil {
		mh.fh.AddError(w, req, err)
	} else {
		mh.fh.AddMessage(w, req, "AuthFallbackPasswordUpdated")
	}

	redirectURL = mh.urlTo(router.AuthFallbackLogin).Path
	http.Redirect(w, req, redirectURL, http.StatusSeeOther)
}
