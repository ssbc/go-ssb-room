package handlers

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"

	"go.mindeco.de/logging"
	"golang.org/x/crypto/ed25519"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	refs "go.mindeco.de/ssb-refs"
)

type inviteHandler struct {
	invites roomdb.InvitesService
	aliases roomdb.AliasesService

	muxrpcHostAndPort string
	roomPubKey        ed25519.PublicKey
}

func (h inviteHandler) acceptForm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	token := req.URL.Query().Get("token")

	inv, err := h.invites.GetByToken(req.Context(), token)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}

	return map[string]interface{}{
		"Token":  token,
		"Invite": inv,

		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h inviteHandler) consume(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	if err := req.ParseForm(); err != nil {
		return nil, weberrors.ErrBadRequest{Where: "form data", Details: err}
	}

	alias := req.FormValue("alias")

	token := req.FormValue("token")

	newMember, err := refs.ParseFeedRef(req.FormValue("new_member"))
	if err != nil {
		return nil, weberrors.ErrBadRequest{Where: "form data", Details: err}
	}

	inv, err := h.invites.Consume(req.Context(), token, *newMember)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}
	log := logging.FromContext(req.Context())
	level.Info(log).Log("event", "invite consumed", "id", inv.ID, "ref", newMember.ShortRef())

	if alias != "" {
		level.Warn(log).Log(
			"TODO", "invite registration",
			"alias", alias,
		)
	}

	// TODO: hardcoded here just to be replaced soon with next version of ssb-uri
	roomPubKey := base64.StdEncoding.EncodeToString(h.roomPubKey)
	roomAddr := fmt.Sprintf("net:%s~shs:%s:SSB+Room+PSK3TLYC2T86EHQCUHBUHASCASE18JBV24=", h.muxrpcHostAndPort, roomPubKey)

	return map[string]interface{}{
		"RoomAddress": roomAddr,
	}, nil
}
