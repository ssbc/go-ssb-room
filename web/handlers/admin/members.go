// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"go.mindeco.de/http/render"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web"
	weberrors "github.com/ssbc/go-ssb-room/v2/web/errors"
	"github.com/ssbc/go-ssb-room/v2/web/members"
	"github.com/ssbc/go-ssb-room/v2/web/router"
)

type membersHandler struct {
	r       *render.Renderer
	flashes *weberrors.FlashHelper
	urlTo   web.URLMaker
	netInfo network.ServerEndpointDetails

	db             roomdb.MembersService
	fallbackAuthDB roomdb.AuthFallbackService
	roomCfgDB      roomdb.RoomConfig
}

const redirectToMembers = "/admin/members"

func (h membersHandler) add(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	if err := req.ParseForm(); err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	defer http.Redirect(w, req, redirectToMembers, http.StatusSeeOther)

	newEntry := req.Form.Get("pub_key")
	newEntryParsed, err := refs.ParseFeedRef(newEntry)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Public Key", Details: err}
		h.flashes.AddError(w, req, err)
		return
	}

	_, err = h.db.Add(req.Context(), newEntryParsed, roomdb.RoleMember)
	if err != nil {
		h.flashes.AddError(w, req, err)
		return
	}

	h.flashes.AddMessage(w, req, "AdminMemberAdded")
}

func (h membersHandler) changeRole(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	if err := req.ParseForm(); err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	currentMember := members.FromContext(req.Context())
	if currentMember == nil || currentMember.Role != roomdb.RoleAdmin {
		err := weberrors.ErrForbidden{Details: fmt.Errorf("not an admin")}
		h.r.Error(w, req, http.StatusForbidden, err)
		return
	}

	memberID, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "id", Details: err}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	var role roomdb.Role
	if err := role.UnmarshalText([]byte(req.Form.Get("role"))); err != nil {
		err = weberrors.ErrBadRequest{Where: "role", Details: err}
		h.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	if err := h.db.SetRole(req.Context(), memberID, role); err != nil {
		err = weberrors.DatabaseError{Reason: err}
		h.r.Error(w, req, http.StatusInternalServerError, err)
		return
	}

	h.flashes.AddMessage(w, req, "AdminMemberUpdated")

	memberDetailsURL := h.urlTo(router.AdminMemberDetails, "id", memberID).String()
	http.Redirect(w, req, memberDetailsURL, http.StatusSeeOther)
}

func (h membersHandler) overview(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
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

	pageData["AllRoles"] = []roomdb.Role{roomdb.RoleMember, roomdb.RoleModerator, roomdb.RoleAdmin}

	pageData["Flashes"], err = h.flashes.GetAll(rw, req)
	if err != nil {
		return nil, err
	}

	return pageData, nil
}

func (h membersHandler) details(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	member, err := h.db.GetByID(req.Context(), id)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	roles := []roomdb.Role{roomdb.RoleMember, roomdb.RoleModerator, roomdb.RoleAdmin}

	aliasURLs := make(map[string]template.URL)
	for _, a := range member.Aliases {
		aliasURLs[a.Name] = template.URL(h.netInfo.URLForAlias(a.Name))
	}

	return map[string]interface{}{
		"Member":         member,
		"AllRoles":       roles,
		"AliasURLs":      aliasURLs,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h membersHandler) removeConfirm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	entry, err := h.db.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			return nil, weberrors.ErrRedirect{
				Path:   redirectToMembers,
				Reason: err,
			}
		}
		return nil, err
	}

	return map[string]interface{}{
		"Entry":          entry,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h membersHandler) remove(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
		h.r.Error(rw, req, http.StatusBadRequest, err)
		return
	}

	if err := req.ParseForm(); err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		h.r.Error(rw, req, http.StatusBadRequest, err)
		return
	}

	defer http.Redirect(rw, req, redirectToMembers, http.StatusSeeOther)

	if _, err := members.CheckAllowed(ctx, h.roomCfgDB, members.ActionRemoveMember); err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		h.flashes.AddError(rw, req, err)
		return
	}

	err = h.db.RemoveID(ctx, id)
	if err != nil {
		h.flashes.AddError(rw, req, err)
	} else {
		h.flashes.AddMessage(rw, req, "AdminMemberRemoved")
	}
}

func (h membersHandler) createPasswordResetToken(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "POST" {
		return nil, weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
	}

	if err := req.ParseForm(); err != nil {
		return nil, weberrors.ErrBadRequest{Where: "Form data", Details: err}
	}

	forMemberID, err := strconv.ParseInt(req.FormValue("member_id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Member ID", Details: err}
		return nil, weberrors.ErrRedirect{Path: redirectToMembers, Reason: err}
	}

	creatingMember := members.FromContext(req.Context())
	if creatingMember == nil || creatingMember.Role != roomdb.RoleAdmin {
		err = weberrors.ErrForbidden{Details: fmt.Errorf("not an admin")}
		return nil, weberrors.ErrRedirect{Path: redirectToMembers, Reason: err}
	}

	token, err := h.fallbackAuthDB.CreateResetToken(req.Context(), creatingMember.ID, forMemberID)
	if err != nil {
		return nil, weberrors.ErrRedirect{Path: redirectToMembers, Reason: err}
	}

	resetFormURL := h.urlTo(router.MembersChangePasswordForm, "token", token)

	return map[string]interface{}{
		"ResetLinkURL": template.URL(resetFormURL.String()),
	}, nil
}
