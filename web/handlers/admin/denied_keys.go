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
	refs "go.mindeco.de/ssb-refs"
)

type deniedKeysHandler struct {
	r *render.Renderer

	flashes *weberrors.FlashHelper

	db      roomdb.DeniedKeysService
	roomCfg roomdb.RoomConfig
}

const redirectToDeniedKeys = "/admin/denied"

func (h deniedKeysHandler) add(w http.ResponseWriter, req *http.Request) {
	// always redirect
	defer http.Redirect(w, req, redirectToDeniedKeys, http.StatusSeeOther)

	ctx := req.Context()

	_, err := members.CheckAllowed(ctx, h.roomCfg, members.ActionChangeDeniedKeys)
	if err != nil {
		err := weberrors.ErrNotAuthorized
		h.flashes.AddError(w, req, err)
		return
	}

	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
		h.flashes.AddError(w, req, err)
		return
	}

	if err := req.ParseForm(); err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		h.flashes.AddError(w, req, err)
		return
	}

	newEntry := req.Form.Get("pub_key")
	newEntryParsed, err := refs.ParseFeedRef(newEntry)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Public Key", Details: err}
		h.flashes.AddError(w, req, err)
		return
	}

	// can be empty
	comment := req.Form.Get("comment")

	err = h.db.Add(req.Context(), *newEntryParsed, comment)
	if err != nil {
		h.flashes.AddError(w, req, err)
	} else {
		h.flashes.AddMessage(w, req, "AdminDeniedKeysAdded")
	}
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
	pageData["Flashes"], err = h.flashes.GetAll(rw, req)
	if err != nil {
		return nil, err
	}

	return pageData, nil
}

func (h deniedKeysHandler) removeConfirm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	entry, err := h.db.GetByID(req.Context(), id)
	if err != nil {
		return nil, weberrors.ErrRedirect{
			Path:   redirectToDeniedKeys,
			Reason: err,
		}
	}

	return map[string]interface{}{
		"Entry":          entry,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h deniedKeysHandler) remove(rw http.ResponseWriter, req *http.Request) {
	// always redirect
	defer http.Redirect(rw, req, redirectToDeniedKeys, http.StatusSeeOther)

	ctx := req.Context()

	_, err := members.CheckAllowed(ctx, h.roomCfg, members.ActionChangeDeniedKeys)
	if err != nil {
		err := weberrors.ErrNotAuthorized
		h.flashes.AddError(rw, req, err)
		return
	}

	err = req.ParseForm()
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

	err = h.db.RemoveID(ctx, id)
	if err != nil {
		h.flashes.AddError(rw, req, err)
	} else {
		h.flashes.AddMessage(rw, req, "AdminDeniedKeysRemoved")
	}
}
