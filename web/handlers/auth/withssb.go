// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"errors"
	"fmt"
	"html/template"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/skip2/go-qrcode"
	"go.cryptoscope.co/muxrpc/v2"
	"go.mindeco.de/http/render"
	kitlog "go.mindeco.de/log"
	"go.mindeco.de/log/level"
	"go.mindeco.de/logging"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
	refs "go.mindeco.de/ssb-refs"
)

var HTMLTemplates = []string{
	"auth/decide_method.tmpl",
	"auth/fallback_sign_in.tmpl",
	"auth/withssb_server_start.tmpl",
}

// custom sessionKey type to prevent collision
type sessionKey uint

func init() {
	// need to register our Key with gob so gorilla/sessions can (de)serialize it
	gob.Register(memberToken)
	gob.Register(time.Time{})
}

const (
	siwssbSessionName = "AuthWithSSBSession"

	memberToken sessionKey = iota
	userTimeout
)

const sessionLifetime = time.Hour * 24

// WithSSBHandler implements the oauth-like challenge/response dance described in
// https://ssb-ngi-pointer.github.io/ssb-http-auth-spec
type WithSSBHandler struct {
	render *render.Renderer

	// roomID            refs.FeedRef
	// muxrpcHostAndPort string

	netInfo network.ServerEndpointDetails
	router  *mux.Router

	membersdb roomdb.MembersService
	aliasesdb roomdb.AliasesService
	sessiondb roomdb.AuthWithSSBService

	cookieStore sessions.Store

	endpoints network.Endpoints

	bridge *signinwithssb.SignalBridge
}

func NewWithSSBHandler(
	m *mux.Router,
	r *render.Renderer,
	netInfo network.ServerEndpointDetails,
	endpoints network.Endpoints,
	aliasDB roomdb.AliasesService,
	membersDB roomdb.MembersService,
	sessiondb roomdb.AuthWithSSBService,
	cookies sessions.Store,
	bridge *signinwithssb.SignalBridge,
) *WithSSBHandler {

	var ssb WithSSBHandler
	ssb.render = r
	ssb.netInfo = netInfo
	ssb.router = m
	ssb.aliasesdb = aliasDB
	ssb.membersdb = membersDB
	ssb.endpoints = endpoints
	ssb.sessiondb = sessiondb
	ssb.cookieStore = cookies
	ssb.bridge = bridge

	m.Get(router.AuthWithSSBLogin).HandlerFunc(ssb.DecideMethod)
	m.Get(router.AuthWithSSBServerEvents).HandlerFunc(ssb.eventSource)
	m.Get(router.AuthWithSSBFinalize).HandlerFunc(ssb.finalizeCookie)

	return &ssb
}

// AuthenticateRequest uses the passed request to load and return the session data that was stored previously.
// If it is invalid or there is no session, it will return ErrNotAuthorized.
// Otherwise it will return the member that belongs to the session.
func (h WithSSBHandler) AuthenticateRequest(r *http.Request) (*roomdb.Member, error) {
	session, err := h.cookieStore.Get(r, siwssbSessionName)
	if err != nil {
		return nil, err
	}

	if session.IsNew {
		return nil, weberrors.ErrNotAuthorized
	}

	tokenVal, ok := session.Values[memberToken]
	if !ok {
		return nil, weberrors.ErrNotAuthorized
	}

	t, ok := session.Values[userTimeout]
	if !ok {
		return nil, weberrors.ErrNotAuthorized
	}

	tout, ok := t.(time.Time)
	if !ok {
		return nil, weberrors.ErrNotAuthorized
	}

	if time.Now().After(tout) {
		return nil, weberrors.ErrNotAuthorized
	}

	token, ok := tokenVal.(string)
	if !ok {
		return nil, weberrors.ErrNotAuthorized
	}

	memberID, err := h.sessiondb.CheckToken(r.Context(), token)
	if err != nil {
		return nil, err
	}

	member, err := h.membersdb.GetByID(r.Context(), memberID)
	if err != nil {
		return nil, err
	}

	return &member, nil
}

// Logout destroys the session data and updates the cookie with an invalidated one.
func (h WithSSBHandler) Logout(w http.ResponseWriter, r *http.Request) error {
	session, err := h.cookieStore.Get(r, siwssbSessionName)
	if err != nil {
		return err
	}

	tokenVal, ok := session.Values[memberToken]
	if !ok {
		// not a ssb http auth session
		return nil
	}

	token, ok := tokenVal.(string)
	if !ok {
		return fmt.Errorf("wrong token type: %T", tokenVal)
	}

	err = h.sessiondb.RemoveToken(r.Context(), token)
	if err != nil {
		return err
	}

	session.Values[userTimeout] = time.Now().Add(-sessionLifetime)
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		return err
	}

	return nil
}

// saveCookie is a utility function that stores the passed token inside the cookie
func (h WithSSBHandler) saveCookie(w http.ResponseWriter, req *http.Request, token string) error {
	session, err := h.cookieStore.Get(req, siwssbSessionName)
	if err != nil {
		err = fmt.Errorf("ssb http auth: failed to load cookie session: %w", err)
		return err
	}

	session.Values[memberToken] = token
	session.Values[userTimeout] = time.Now().Add(sessionLifetime)
	if err := session.Save(req, w); err != nil {
		err = fmt.Errorf("ssb http auth: failed to update cookie session: %w", err)
		return err
	}

	return nil
}

// this is the /login landing page which branches out to the different methods
// based on the query parameters that are present
func (h WithSSBHandler) DecideMethod(w http.ResponseWriter, req *http.Request) {
	queryVals := req.URL.Query()

	var (
		alias string = queryVals.Get("alias")
		cid   *refs.FeedRef
	)

	if cidString := queryVals.Get("cid"); cidString != "" {
		parsedCID, err := refs.ParseFeedRef(cidString)
		if err == nil {
			cid = parsedCID

			_, err := h.membersdb.GetByFeed(req.Context(), *cid)
			if err != nil {
				if errors.Is(err, roomdb.ErrNotFound) {
					h.render.Error(w, req, http.StatusForbidden, weberrors.ErrForbidden{Details: err})
					return
				}
				h.render.Error(w, req, http.StatusInternalServerError, err)
				return
			}
		}
	} else {
		aliasEntry, err := h.aliasesdb.Resolve(req.Context(), alias)
		if err == nil {
			cid = &aliasEntry.Feed
		}
	}

	// ?cid=CID&cc=CC does client-initiated http-auth
	if cc := queryVals.Get("cc"); cc != "" && cid != nil {
		err := h.clientInitiated(w, req, *cid)
		if err != nil {
			h.render.Error(w, req, http.StatusInternalServerError, err)
		}
		return
	}

	// assume server-init sse dance
	sc := queryVals.Get("sc") // is non-empty when a remote device sends the solution
	data, err := h.serverInitiated(sc)
	if err != nil {
		h.render.Error(w, req, http.StatusInternalServerError, err)
		return
	}
	h.render.Render(w, req, "auth/withssb_server_start.tmpl", http.StatusOK, data)
}

// clientInitiated is called with a client challange (?cc=123) and calls back to
// the passed client using muxrpc to request a signed solution.
// if everything checks out it redirects to the admin dashboard
func (h WithSSBHandler) clientInitiated(w http.ResponseWriter, req *http.Request, client refs.FeedRef) error {
	queryParams := req.URL.Query()

	var payload signinwithssb.ClientPayload
	payload.ServerID = h.netInfo.RoomID // fill in the server

	// validate and update client challenge
	cc := queryParams.Get("cc")
	payload.ClientChallenge = cc

	// check that we have that member
	member, err := h.membersdb.GetByFeed(req.Context(), client)
	if err != nil {
		if errors.Is(err, roomdb.ErrNotFound) {
			errMsg := fmt.Errorf("ssb http auth: client isn't a member: %w", err)
			return weberrors.ErrForbidden{Details: errMsg}
		}
		return err
	}
	payload.ClientID = client

	// get the connected client for that member
	edp, connected := h.endpoints.GetEndpointFor(client)
	if !connected {
		return weberrors.ErrForbidden{Details: fmt.Errorf("ssb http auth: client not connected to room")}
	}

	// roll a Challenge from the server
	sc := signinwithssb.GenerateChallenge()
	payload.ServerChallenge = sc

	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()

	// request the signed solution over muxrpc
	var solution string
	err = edp.Async(ctx, &solution, muxrpc.TypeString, muxrpc.Method{"httpAuth", "requestSolution"}, sc, cc)
	if err != nil {
		return fmt.Errorf("ssb http auth: could not request solution from client: %w", err)
	}

	// decode and validate the response
	solution = strings.TrimSuffix(solution, ".sig.ed25519")
	solutionBytes, err := base64.StdEncoding.DecodeString(solution)
	if err != nil {
		return fmt.Errorf("ssb http auth: failed to decode solution: %w", err)
	}

	if !payload.Validate(solutionBytes) {
		return fmt.Errorf("ssb http auth: validation of client solution failed")
	}

	// create a session for invalidation
	tok, err := h.sessiondb.CreateToken(req.Context(), member.ID)
	if err != nil {
		err = fmt.Errorf("ssb http auth: could not create token: %w", err)
		return err
	}

	if err := h.saveCookie(w, req, tok); err != nil {
		return err
	}

	// go to the dashboard
	dashboardURL, err := router.CompleteApp().Get(router.AdminDashboard).URL()
	if err != nil {
		return err
	}

	http.Redirect(w, req, dashboardURL.Path, http.StatusTemporaryRedirect)
	return nil
}

// server-sent-events stuff

type templateData struct {
	SSBURI            template.URL
	QRCodeURI         template.URL
	IsSolvingRemotely bool
	ServerChallenge   string
}

func (h WithSSBHandler) serverInitiated(sc string) (templateData, error) {
	isSolvingRemotely := true
	if sc == "" {
		isSolvingRemotely = false
		sc = h.bridge.RegisterSession()
	}

	// prepare the ssb-uri
	// https://ssb-ngi-pointer.github.io/ssb-http-auth-spec/#list-of-new-ssb-uris
	var queryParams = make(url.Values)
	queryParams.Set("action", "start-http-auth")
	queryParams.Set("sid", h.netInfo.RoomID.Ref())
	queryParams.Set("sc", sc)
	queryParams.Set("multiserverAddress", h.netInfo.MultiserverAddress())

	var startAuthURI url.URL
	startAuthURI.Scheme = "ssb"
	startAuthURI.Opaque = "experimental"
	startAuthURI.RawQuery = queryParams.Encode()

	var qrURI string
	if !isSolvingRemotely {
		urlTo := web.NewURLTo(router.Auth(h.router), h.netInfo)
		remoteLoginURL := urlTo(router.AuthWithSSBLogin, "sc", sc)
		remoteLoginURL.Host = h.netInfo.Domain
		remoteLoginURL.Scheme = "https"
		if h.netInfo.Development {
			remoteLoginURL.Scheme = "http"
			remoteLoginURL.Host += fmt.Sprintf(":%d", h.netInfo.PortHTTPS)
		}

		// generate a QR code with the login URL inside so that you can open it
		// easily in a supporting mobile app
		qrCode, err := qrcode.New(remoteLoginURL.String(), qrcode.Medium)
		if err != nil {
			return templateData{}, err
		}

		qrCode.BackgroundColor = color.Transparent // transparent to fit into the page
		qrCode.ForegroundColor = color.Black

		qrCodeData, err := qrCode.PNG(-5)
		if err != nil {
			return templateData{}, err
		}
		qrURI = "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrCodeData)
	}

	// template.URL signals the template engine that those aren't fishy and from a trusted source

	data := templateData{
		SSBURI:            template.URL(startAuthURI.String()),
		QRCodeURI:         template.URL(qrURI),
		IsSolvingRemotely: isSolvingRemotely,

		ServerChallenge: sc,
	}
	return data, nil
}

// finalizeCookie is called with a redirect from the js sse client if everything worked
func (h WithSSBHandler) finalizeCookie(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")

	// check the token is correct
	if _, err := h.sessiondb.CheckToken(r.Context(), tok); err != nil {
		http.Error(w, "invalid session token", http.StatusForbidden)
		return
	}

	if err := h.saveCookie(w, r, tok); err != nil {
		http.Error(w, "failed to save cookie", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// the time after which the SSE dance is considered failed
const sseTimeout = 3 * time.Minute

// eventSource is the server-side of our server-sent events (SSE) session
// https://html.spec.whatwg.org/multipage/server-sent-events.html
func (h WithSSBHandler) eventSource(w http.ResponseWriter, r *http.Request) {
	flusher, err := w.(http.Flusher)
	if !err {
		http.Error(w, "ssb http auth: server-initiated method needs streaming support", http.StatusInternalServerError)
		return
	}

	// closes when the http request is closed
	var notify <-chan bool

	notifier, ok := w.(http.CloseNotifier)
	if !ok {
		// testing hack
		// http.Error(w, "ssb http auth: cant notify about closed requests", http.StatusInternalServerError)
		// return
		ch := make(chan bool)
		go func() {
			time.Sleep(sseTimeout)
			close(ch)
		}()
		notify = ch
	} else {
		notify = notifier.CloseNotify()
	}

	// setup headers for SSE
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	sc := r.URL.Query().Get("sc")
	if sc == "" {
		http.Error(w, "missing server challenge", http.StatusBadRequest)
		return
	}

	logger := logging.FromContext(r.Context())
	logger = level.Debug(logger)
	logger = kitlog.With(logger, "stream", sc[:5])
	logger.Log("event", "stream opened")

	evtCh, has := h.bridge.GetEventChannel(sc)
	if !has {
		http.Error(w, "no such session!", http.StatusBadRequest)
		return
	}

	sender := newEventSender(w)

	// ping ticker
	tick := time.NewTicker(3 * time.Second)
	go func() {
		time.Sleep(sseTimeout)
		tick.Stop()
		logger.Log("event", "stopped")
	}()

	start := time.Now()
	flusher.Flush()

	// Push events to client

	for {
		select {

		case <-notify:
			logger.Log("event", "request closed")
			return

		case <-tick.C:
			sender.send("ping", fmt.Sprintf("Waiting for solution (session age: %s)", time.Since(start)))
			logger.Log("event", "sent ping")

		case update := <-evtCh:
			var event, data string = "failed", "challenge validation failed"

			if update.Worked {
				event = "success"
				data = update.Token
			} else {
				if update.Reason != nil {
					data = update.Reason.Error()
				}
			}

			sender.send(event, data)
			logger.Log("event", "sent", "worked", update.Worked)
			return
		}

		flusher.Flush()
	}
}

// eventSender encapsulates the event ID and increases it with each send automatically
type eventSender struct {
	w io.Writer

	id uint32
}

func newEventSender(w io.Writer) eventSender {
	return eventSender{w: w}
}

func (es *eventSender) send(event, data string) {
	fmt.Fprintf(es.w, "id: %d\n", es.id)
	fmt.Fprintf(es.w, "data: %s\n", data)
	fmt.Fprintf(es.w, "event: %s\n", event)
	fmt.Fprint(es.w, "\n")
	es.id++
}
