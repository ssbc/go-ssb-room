package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"github.com/gorilla/csrf"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

type invitesHandler struct {
	r       *render.Renderer
	flashes *weberrors.FlashHelper
	urlTo   web.URLMaker

	db     roomdb.InvitesService
	config roomdb.RoomConfig

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
	pageData["Flashes"], err = h.flashes.GetAll(rw, req)
	if err != nil {
		return nil, err
	}
	return pageData, nil
}

func (h invitesHandler) create(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	if req.Method != "POST" {
		return nil, weberrors.ErrBadRequest{Where: "HTTP Method", Details: fmt.Errorf("expected POST not %s", req.Method)}
	}
	if err := req.ParseForm(); err != nil {
		return nil, weberrors.ErrBadRequest{Where: "Form data", Details: err}
	}

	member := members.FromContext(req.Context())
	if member == nil {
		return nil, weberrors.ErrNotAuthorized
	}
	pm, err := h.config.GetPrivacyMode(req.Context())
	if err != nil {
		return nil, err
	}
	/* We want to check:
		 * 1. the room's privacy mode
		 * 2. the role of the member trying to create the invite
	     * and deny unallowed requests (e.g. member creating invite in ModeRestricted)
	*/
	switch pm {
	case roomdb.ModeOpen:
	case roomdb.ModeCommunity:
		if member.Role == roomdb.RoleUnknown {
			return nil, weberrors.ErrNotAuthorized
		}
	case roomdb.ModeRestricted:
		if member.Role == roomdb.RoleMember || member.Role == roomdb.RoleUnknown {
			return nil, weberrors.ErrNotAuthorized
		}
	}

	token, err := h.db.Create(req.Context(), member.ID)
	if err != nil {
		return nil, err
	}

	facadeURL := h.urlTo(router.CompleteInviteFacade, "token", token)
	facadeURL.Host = h.domainName
	facadeURL.Scheme = "https"

	return map[string]interface{}{
		"FacadeURL": facadeURL.String(),
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
		if errors.Is(err, roomdb.ErrNotFound) {
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
		if !errors.Is(err, roomdb.ErrNotFound) {
			// TODO "flash" errors
			h.r.Error(rw, req, http.StatusInternalServerError, err)
			return
		}
		status = http.StatusNotFound
	}

	http.Redirect(rw, req, redirectToInvites, status)
}
