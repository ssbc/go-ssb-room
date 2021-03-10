package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/user"
)

type invitesHandler struct {
	r *render.Renderer

	db admindb.InviteService

	domainName string
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
	return pageData, nil
}

func (h invitesHandler) create(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "POST" {
		// TODO: proper error type
		return nil, fmt.Errorf("bad request")
	}
	if err := req.ParseForm(); err != nil {
		// TODO: proper error type
		return nil, fmt.Errorf("bad request: %w", err)
	}

	user := user.FromContext(req.Context())
	if user == nil {
		return nil, fmt.Errorf("warning: no user session for elevated access request")
	}

	aliasSuggestion := req.Form.Get("alias_suggestion")

	token, err := h.db.Create(req.Context(), user.ID, aliasSuggestion)
	if err != nil {
		return nil, err
	}

	urlTo := web.NewURLTo(router.CompleteApp())
	acceptURL := urlTo(router.CompleteInviteAccept, "token", token)
	acceptURL.Host = h.domainName
	acceptURL.Scheme = "https"

	return map[string]interface{}{
		"Token":    token,
		"AccepURL": acceptURL.String(),

		"AliasSuggestion": aliasSuggestion,
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
		if errors.Is(err, admindb.ErrNotFound) {
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
	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "Form data", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectToInvites, http.StatusFound)
		return
	}

	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectToInvites, http.StatusFound)
		return
	}

	status := http.StatusFound
	err = h.db.Revoke(req.Context(), id)
	if err != nil {
		if !errors.Is(err, admindb.ErrNotFound) {
			// TODO "flash" errors
			h.r.Error(rw, req, http.StatusInternalServerError, err)
			return
		}
		status = http.StatusNotFound
	}

	http.Redirect(rw, req, redirectToInvites, status)
}
