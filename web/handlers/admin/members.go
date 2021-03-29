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

	db roomdb.MembersService
}

const redirectToMembers = "/admin/members"

func (h membersHandler) add(w http.ResponseWriter, req *http.Request) {
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

	newEntry := req.Form.Get("pub_key")
	newEntryParsed, err := refs.ParseFeedRef(newEntry)
	if err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad public key: %w", err))
		return
	}

	_, err = h.db.Add(req.Context(), *newEntryParsed, roomdb.RoleMember)
	if err != nil {
		code := http.StatusInternalServerError
		var aa roomdb.ErrAlreadyAdded
		if errors.As(err, &aa) {
			code = http.StatusBadRequest
			// TODO: localized error pages
			// h.r.Error(w, req, http.StatusBadRequest, weberrors.Localize())
			// return
		}

		h.r.Error(w, req, code, err)
		return
	}

	http.Redirect(w, req, redirectToMembers, http.StatusFound)
}

func (h membersHandler) changeRole(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request"))
		return
	}

	currentMember := members.FromContext(req.Context())
	if currentMember == nil || currentMember.Role != roomdb.RoleAdmin {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusForbidden, fmt.Errorf("not an admin"))
		return
	}

	memberID, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad member id: %w", err))
		return
	}

	if err := req.ParseForm(); err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request: %w", err))
		return
	}

	var role roomdb.Role
	if err := role.UnmarshalText([]byte(req.Form.Get("role"))); err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, err)
		return
	}

	if err := h.db.SetRole(req.Context(), memberID, role); err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusInternalServerError, fmt.Errorf("failed to change member role: %w", err))
		return
	}

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
			http.Redirect(rw, req, redirectToMembers, http.StatusFound)
			return nil, ErrRedirected
		}
		return nil, err
	}

	return map[string]interface{}{
		"Entry":          entry,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h membersHandler) remove(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectToMembers, http.StatusFound)
		return
	}

	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectToMembers, http.StatusFound)
		return
	}

	status := http.StatusFound
	err = h.db.RemoveID(req.Context(), id)
	if err != nil {
		if !errors.Is(err, roomdb.ErrNotFound) {
			// TODO "flash" errors
			h.r.Error(rw, req, http.StatusInternalServerError, err)
			return
		}
		status = http.StatusNotFound
	}

	http.Redirect(rw, req, redirectToMembers, status)
}
