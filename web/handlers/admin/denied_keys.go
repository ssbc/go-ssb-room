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
)

type deniedKeysHandler struct {
	r *render.Renderer

	db roomdb.DeniedKeysService
}

const redirectToDeniedKeys = "/admin/denied"

func (h deniedKeysHandler) add(w http.ResponseWriter, req *http.Request) {
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
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request: %w", err))
		return
	}

	// can be empty
	comment := req.Form.Get("comment")

	err = h.db.Add(req.Context(), *newEntryParsed, comment)
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

	http.Redirect(w, req, redirectToDeniedKeys, http.StatusFound)
}

func (h deniedKeysHandler) overview(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
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

	return pageData, nil
}

// TODO: move to render package so that we can decide to not render a page during the controller
var ErrRedirected = errors.New("render: not rendered but redirected")

func (h deniedKeysHandler) removeConfirm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	entry, err := h.db.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			http.Redirect(rw, req, redirectToDeniedKeys, http.StatusFound)
			return nil, ErrRedirected
		}
		return nil, err
	}

	return map[string]interface{}{
		"Entry":          entry,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h deniedKeysHandler) remove(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectToDeniedKeys, http.StatusFound)
		return
	}

	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectToDeniedKeys, http.StatusFound)
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

	http.Redirect(rw, req, redirectToDeniedKeys, status)
}
