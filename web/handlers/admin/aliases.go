// SPDX-License-Identifier: MIT

package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
)

// aliasesHandler implements the managment endpoints for aliases (list and revoke),
// does light validation of the web arguments and passes them through to the roomdb.
type aliasesHandler struct {
	r *render.Renderer

	db roomdb.AliasesService
}

const redirectToAliases = "/admin/aliases"

func (h aliasesHandler) overview(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
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

	return pageData, nil
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
		if errors.Is(err, roomdb.ErrNotFound) {
			http.Redirect(rw, req, redirectToAliases, http.StatusFound)
			return nil, ErrRedirected
		}
		return nil, err
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
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		http.Redirect(rw, req, redirectToAliases, http.StatusFound)
		return
	}

	status := http.StatusFound
	err = h.db.Revoke(req.Context(), req.FormValue("name"))
	if err != nil {
		if !errors.Is(err, roomdb.ErrNotFound) {
			//  TODO: flash error
			h.r.Error(rw, req, http.StatusInternalServerError, err)
			return
		}
		status = http.StatusNotFound
		h.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(rw, req, redirectToAliases, status)
}
