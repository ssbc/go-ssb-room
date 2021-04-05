// SPDX-License-Identifier: MIT

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"go.mindeco.de/http/render"
	refs "go.mindeco.de/ssb-refs"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/members"
)

type membersHandler struct {
	r *render.Renderer

	flashes *weberrors.FlashHelper

	db roomdb.MembersService
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

	newEntry := req.Form.Get("pub_key")
	newEntryParsed, err := refs.ParseFeedRef(newEntry)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Public Key", Details: err}
		h.flashes.AddError(w, req, err)
		http.Redirect(w, req, redirectToMembers, http.StatusTemporaryRedirect)
		return
	}

	_, err = h.db.Add(req.Context(), *newEntryParsed, roomdb.RoleMember)
	if err != nil {
		h.flashes.AddError(w, req, err)
	} else {
		h.flashes.AddMessage(w, req, "AdminMemberAdded")
	}

	http.Redirect(w, req, redirectToMembers, http.StatusTemporaryRedirect)
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
	http.Redirect(w, req, redirectToMembers, http.StatusTemporaryRedirect)
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

	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		h.flashes.AddError(rw, req, err)
		http.Redirect(rw, req, redirectToMembers, http.StatusTemporaryRedirect)
		return
	}

	err = h.db.RemoveID(req.Context(), id)
	if err != nil {
		h.flashes.AddError(rw, req, err)
	} else {
		h.flashes.AddMessage(rw, req, "AdminMemberRemoved")
	}

	http.Redirect(rw, req, redirectToMembers, http.StatusTemporaryRedirect)
}
