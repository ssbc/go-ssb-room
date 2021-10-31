// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
)

type invitesHandler struct {
	r       *render.Renderer
	flashes *weberrors.FlashHelper
	urlTo   web.URLMaker

	db     roomdb.InvitesService
	config roomdb.RoomConfig
}

func (h invitesHandler) overview(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	lst, err := h.db.List(req.Context())
	if err != nil {
		return nil, err
	}

	// Reverse the slice to provide recent-to-oldest results
	for i, j := 0, len(lst)-1; i < j; i, j = i+1, j-1 {
		lst[i], lst[j] = lst[j], lst[i]
	}

	pageData, err := paginate(lst, len(lst), req.URL.Query())
	if err != nil {
		return nil, err
	}

	pageData[csrf.TemplateTag] = csrf.TemplateField(req)
	pageData["Flashes"], err = h.flashes.GetAll(rw, req)
	if err != nil {
		return nil, err
	}
	return pageData, nil
}

func (h invitesHandler) create(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "POST" {
		return nil, weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
	}

	if err := req.ParseForm(); err != nil {
		return nil, weberrors.ErrBadRequest{Where: "Form data", Details: err}
	}

	ctx := req.Context()

	member, err := members.CheckAllowed(ctx, h.config, members.ActionInviteMember)
	if err != nil {
		return nil, err
	}

	token, err := h.db.Create(ctx, member.ID)
	if err != nil {
		return nil, err
	}

	facadeURL := h.urlTo(router.CompleteInviteFacade, "token", token)

	return map[string]interface{}{
		"FacadeURL": facadeURL.String(),
	}, nil
}

func (h invitesHandler) revokeConfirm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	invite, err := h.db.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}

	return map[string]interface{}{
		"Invite":         invite,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

const redirectToInvites = "/admin/invites"

func (h invitesHandler) revoke(rw http.ResponseWriter, req *http.Request) {
	// always redirect
	defer http.Redirect(rw, req, redirectToInvites, http.StatusSeeOther)

	ctx := req.Context()

	if _, err := members.CheckAllowed(ctx, h.config, members.ActionInviteMember); err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		h.flashes.AddError(rw, req, err)
		return
	}

	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		h.flashes.AddError(rw, req, err)
		return
	}

	err = h.db.Revoke(ctx, id)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			err = weberrors.ErrNotFound{What: "invite"}
		}
		h.flashes.AddError(rw, req, err)
	} else {
		h.flashes.AddMessage(rw, req, "InviteRevoked")
	}
}
