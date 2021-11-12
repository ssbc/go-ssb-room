// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"image/color"
	"net/http"
	"net/url"

	"github.com/gorilla/csrf"
	"github.com/skip2/go-qrcode"
	"go.mindeco.de/http/render"
	"go.mindeco.de/log/level"
	"go.mindeco.de/logging"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
	refs "go.mindeco.de/ssb-refs"
)

type inviteHandler struct {
	render      *render.Renderer
	urlTo       web.URLMaker
	networkInfo network.ServerEndpointDetails

	invites       roomdb.InvitesService
	pinnedNotices roomdb.PinnedNoticesService
	config        roomdb.RoomConfig
	deniedKeys    roomdb.DeniedKeysService
}

func (h inviteHandler) buildJoinRoomURI(token string, userAgent string) template.URL {
	queryVals := make(url.Values)
	queryVals.Set("action", "claim-http-invite")
	queryVals.Set("invite", token)

	submissionURL := h.urlTo(router.CompleteInviteConsume)
	queryVals.Set("postTo", submissionURL.String())

	joinRoomURI := url.URL{
		Scheme:   "ssb",
		Opaque:   "experimental",
		RawQuery: queryVals.Encode(),
	}

	return template.URL(web.StringifySSBURI(&joinRoomURI, userAgent))
}

// switch between JSON and HTML responses
func (h inviteHandler) presentFacade(rw http.ResponseWriter, req *http.Request) {
	enc := req.URL.Query().Get("encoding")
	if enc == "json" {
		h.presentFacadeAsJSON(rw, req)
		return
	}

	h.render.HTML("invite/facade.tmpl", h.presentFacadeAsHTML)(rw, req)
}

func (h inviteHandler) presentFacadeAsJSON(rw http.ResponseWriter, req *http.Request) {
	logger := logging.FromContext(req.Context())

	rw.Header().Set("Content-Type", "application/json")
	rw.WriteHeader(http.StatusOK)

	enc := json.NewEncoder(rw)

	// make sure token is still valid
	token := req.URL.Query().Get("token")
	_, err := h.invites.GetByToken(req.Context(), token)
	if err != nil {
		// send a json error back
		data := struct {
			Status string `json:"status"`
			Error  string `json:"error"`
		}{"failed", err.Error()}
		if err := enc.Encode(data); err != nil {
			level.Warn(logger).Log("event", "sending json error failed", "err", err)
		}
		return
	}

	// send them on to the next step
	postTo := h.urlTo(router.CompleteInviteConsume)
	data := struct {
		Status string `json:"status"`
		Invite string `json:"invite"`
		PostTo string `json:"postTo"`
	}{"success", token, postTo.String()}
	if err := enc.Encode(data); err != nil {
		level.Warn(logger).Log("event", "sending json response failed", "err", err)
	}
}

func (h inviteHandler) presentFacadeAsHTML(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	token := req.URL.Query().Get("token")

	_, err := h.invites.GetByToken(req.Context(), token)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}

	notice, err := h.pinnedNotices.Get(req.Context(), roomdb.NoticeDescription, "en-GB")
	if err != nil {
		return nil, fmt.Errorf("failed to find room's description: %w", err)
	}

	joinRoomURI := h.buildJoinRoomURI(token, req.UserAgent())

	fallbackURL := h.urlTo(router.CompleteInviteFacadeFallback, "token", token)

	// generate a QR code with the token inside so that you can open it easily in a supporting mobile app
	thisURL := req.URL
	thisURL.Host = h.networkInfo.Domain
	thisURL.Scheme = "https"
	if h.networkInfo.Development {
		thisURL.Scheme = "http"
		thisURL.Host += fmt.Sprintf(":%d", h.networkInfo.PortHTTPS)
	}
	qrCode, err := qrcode.New(thisURL.String(), qrcode.Medium)
	if err != nil {
		return nil, err
	}

	qrCode.BackgroundColor = color.Transparent // transparent to fit into the page
	qrCode.ForegroundColor = color.Black

	qrCodeData, err := qrCode.PNG(-5)
	if err != nil {
		return nil, err
	}
	qrURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrCodeData)

	return map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
		"RoomTitle":      notice.Title,
		"JoinRoomURI":    joinRoomURI,
		"FallbackURL":    fallbackURL,
		"QRCodeURI":      template.URL(qrURI),
	}, nil
}

func (h inviteHandler) presentFacadeFallback(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	token := req.URL.Query().Get("token")

	_, err := h.invites.GetByToken(req.Context(), token)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}

	insertURL := h.urlTo(router.CompleteInviteInsertID, "token", token)

	return map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
		"InsertURL":      insertURL,
	}, nil
}

func (h inviteHandler) presentInsert(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	token := req.URL.Query().Get("token")

	_, err := h.invites.GetByToken(req.Context(), token)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			return nil, weberrors.ErrNotFound{What: "invite"}
		}
		return nil, err
	}

	return map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
		"Token":          token,
	}, nil
}

type inviteConsumePayload struct {
	ID     refs.FeedRef `json:"id"`
	Invite string       `json:"invite"`
}

func (h inviteHandler) consume(rw http.ResponseWriter, req *http.Request) {
	logger := logging.FromContext(req.Context())

	var (
		token     string
		newMember refs.FeedRef
		resp      inviteConsumeResponder
	)

	ct := req.Header.Get("Content-Type")
	switch ct {
	case "application/json":
		resp = newinviteConsumeJSONResponder(rw)

		var body inviteConsumePayload

		level.Debug(logger).Log("event", "handling json body")
		err := json.NewDecoder(req.Body).Decode(&body)
		if err != nil {
			err = fmt.Errorf("consume body contained invalid json: %w", err)
			resp.SendError(err)
			return
		}

		newMember = body.ID
		token = body.Invite
	case "application/x-www-form-urlencoded":
		resp = newinviteConsumeHTMLResponder(h.render, rw, req)

		if err := req.ParseForm(); err != nil {
			err = weberrors.ErrBadRequest{Where: "form data", Details: err}
			resp.SendError(err)
			return
		}

		token = req.FormValue("invite")

		parsedID, err := refs.ParseFeedRef(req.FormValue("id"))
		if err != nil {
			err = weberrors.ErrBadRequest{Where: "id", Details: err}
			resp.SendError(err)
			return
		}
		newMember = *parsedID
	default:
		http.Error(rw, fmt.Sprintf("unhandled Content-Type (%q)", ct), http.StatusBadRequest)
		return
	}

	// before consuming the invite: check if the invitee is banned
	if h.deniedKeys.HasFeed(req.Context(), newMember) {
		resp.SendError(weberrors.ErrDenied)
		return
	}

	resp.UpdateMultiserverAddr(h.networkInfo.MultiserverAddress())

	inv, err := h.invites.Consume(req.Context(), token, newMember)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			resp.SendError(weberrors.ErrNotFound{What: "invite"})
			return
		}
		resp.SendError(err)
		return
	}
	log := logging.FromContext(req.Context())
	level.Info(log).Log("event", "invite consumed", "id", inv.ID, "ref", newMember.ShortRef())

	resp.SendSuccess()
}

// inviteConsumeResponder is supposed to handle different encoding types transparently.
// It either sends the rooms multiaddress on success or an error.
type inviteConsumeResponder interface {
	SendSuccess()
	SendError(error)

	UpdateMultiserverAddr(string)
}

// inviteConsumeJSONResponse dictates the field names and format of the JSON response for the inviteConsume web endpoint
type inviteConsumeJSONResponse struct {
	Status string `json:"status"`

	RoomAddress string `json:"multiserverAddress"`
}

// handles JSON responses
type inviteConsumeJSONResponder struct {
	enc *json.Encoder

	multiservAddr string
}

func newinviteConsumeJSONResponder(rw http.ResponseWriter) inviteConsumeResponder {
	rw.Header().Set("Content-Type", "application/json")
	return &inviteConsumeJSONResponder{
		enc: json.NewEncoder(rw),
	}
}

func (json *inviteConsumeJSONResponder) UpdateMultiserverAddr(msaddr string) {
	json.multiservAddr = msaddr
}

func (json inviteConsumeJSONResponder) SendSuccess() {
	var resp = inviteConsumeJSONResponse{
		Status:      "successful",
		RoomAddress: json.multiservAddr,
	}
	json.enc.Encode(resp)
}

func (json inviteConsumeJSONResponder) SendError(err error) {
	json.enc.Encode(struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}{"error", err.Error()})
}

// handles HTML responses
type inviteConsumeHTMLResponder struct {
	renderer *render.Renderer
	rw       http.ResponseWriter
	req      *http.Request

	multiservAddr string
}

func newinviteConsumeHTMLResponder(r *render.Renderer, rw http.ResponseWriter, req *http.Request) inviteConsumeResponder {
	return &inviteConsumeHTMLResponder{
		renderer: r,
		rw:       rw,
		req:      req,
	}
}

func (html *inviteConsumeHTMLResponder) UpdateMultiserverAddr(msaddr string) {
	html.multiservAddr = msaddr
}

func (html inviteConsumeHTMLResponder) SendSuccess() {
	err := html.renderer.Render(html.rw, html.req, "invite/consumed.tmpl", http.StatusOK, struct {
		MultiserverAddress string
	}{(html.multiservAddr)})
	if err != nil {
		logger := logging.FromContext(html.req.Context())
		level.Warn(logger).Log("event", "render failed", "err", err)
	}
}

func (html inviteConsumeHTMLResponder) SendError(err error) {
	html.renderer.Error(html.rw, html.req, http.StatusInternalServerError, err)
}

func (h inviteHandler) createOpenMode(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	ctx := req.Context()

	token, err := h.invites.Create(ctx, -1)
	if err != nil {
		return nil, err
	}

	facadeURL := h.urlTo(router.CompleteInviteFacade, "token", token)

	return map[string]interface{}{
		"FacadeURL": facadeURL.String(),
	}, nil
}
