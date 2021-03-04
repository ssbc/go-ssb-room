package handlers

import (
	"errors"
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	refs "go.mindeco.de/ssb-refs"
)

type inviteHandler struct {
	r *render.Renderer

	invites admindb.InviteService
	alaises admindb.AliasService
}

func (h inviteHandler) acceptForm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	inv, err := h.invites.GetByToken(req.Context(), req.URL.Query().Get("token"))
	if err != nil {
		if errors.Is(err, admindb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}

	return map[string]interface{}{
		"Invite":         inv,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h inviteHandler) consume(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	if err := req.ParseForm(); err != nil {
		return nil, weberrors.ErrBadRequest{Where: "form data", Details: err}
	}

	token := req.FormValue("token")

	newMember, err := refs.ParseFeedRef(req.FormValue("new_member"))
	if err != nil {
		return nil, weberrors.ErrBadRequest{Where: "form data", Details: err}
	}

	inv, err := h.invites.Consume(req.Context(), token, *newMember)
	if err != nil {
		if errors.Is(err, admindb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}

	return map[string]interface{}{
		"TunnelAddress": "pew pew",
	}, nil
}
