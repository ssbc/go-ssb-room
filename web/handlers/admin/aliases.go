// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/members"
)

// aliasesHandler implements the managment endpoints for aliases (list and revoke),
// does light validation of the web arguments and passes them through to the roomdb.
type aliasesHandler struct {
	r *render.Renderer

	flashes *weberrors.FlashHelper

	db roomdb.AliasesService
}

func (h aliasesHandler) revokeConfirm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "GET" {
		return nil, weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected GET request")}
	}

	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	entry, err := h.db.GetByID(req.Context(), id)
	if err != nil {
		return nil, weberrors.ErrRedirect{
			Path:   redirectToMembers,
			Reason: err,
		}
	}

	return map[string]interface{}{
		"Entry":          entry,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h aliasesHandler) revoke(rw http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST request")}
		h.r.Error(rw, req, http.StatusMethodNotAllowed, err)
		return
	}

	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrRedirect{
			Path:   redirectToMembers,
			Reason: weberrors.ErrBadRequest{Where: "Form data", Details: err},
		}
		h.r.Error(rw, req, http.StatusBadRequest, err)
		return
	}

	defer http.Redirect(rw, req, redirectToMembers, http.StatusSeeOther)

	aliasName := req.FormValue("name")

	ctx := req.Context()

	aliasEntry, err := h.db.Resolve(ctx, aliasName)
	if err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	// who is doing this request
	currentMember := members.FromContext(ctx)
	if currentMember == nil {
		err := weberrors.ErrForbidden{Details: fmt.Errorf("not an member")}
		h.flashes.AddError(rw, req, err)
		return
	}

	// ensure own alias or admin
	if !aliasEntry.Feed.Equal(&currentMember.PubKey) && currentMember.Role != roomdb.RoleAdmin {
		err := weberrors.ErrForbidden{Details: fmt.Errorf("not your alias or not an admin")}
		h.flashes.AddError(rw, req, err)
		return
	}

	err = h.db.Revoke(ctx, aliasName)
	if err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	h.flashes.AddMessage(rw, req, "AdminMemberDetailsAliasRevoked")
}
