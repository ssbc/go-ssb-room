// SPDX-License-Identifier: MIT

package admin

import (
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

	flashes *weberrors.FlashHelper

	db roomdb.AliasesService
}

const redirectToAliases = "/admin/aliases"

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
		h.flashes.AddError(rw, req, err)
		return nil, weberrors.ErrRedirect{Path: redirectToAliases}
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

	status := http.StatusTemporaryRedirect
	err = h.db.Revoke(req.Context(), req.FormValue("name"))
	if err != nil {
		h.flashes.AddError(rw, req, err)
	} else {
		h.flashes.AddMessage(rw, req, "AdminAliasRevoked")
	}

	http.Redirect(rw, req, redirectToAliases, status)
}
