// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/base64"
	"encoding/gob"
	"fmt"
	"html/template"
	"image/color"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/skip2/go-qrcode"
	"go.cryptoscope.co/muxrpc/v2"
	"go.mindeco.de/http/render"
	"go.mindeco.de/logging"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	refs "go.mindeco.de/ssb-refs"
)

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

	roomID refs.FeedRef

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
	roomID refs.FeedRef,
	endpoints network.Endpoints,
	aliasDB roomdb.AliasesService,
	membersDB roomdb.MembersService,
	sessiondb roomdb.AuthWithSSBService,
	cookies sessions.Store,
	bridge *signinwithssb.SignalBridge,
) *WithSSBHandler {

	var ssb WithSSBHandler
	ssb.render = r
	ssb.roomID = roomID
	ssb.aliasesdb = aliasDB
	ssb.membersdb = membersDB
	ssb.endpoints = endpoints
	ssb.sessiondb = sessiondb
	ssb.cookieStore = cookies
	ssb.bridge = bridge

	m.Get(router.AuthLogin).HandlerFunc(ssb.decideMethod)

	m.HandleFunc("/sse/events", ssb.eventSource)
	m.HandleFunc("/sse/finalize", ssb.finalizeCookie)

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

// this is the /login landing page which branches out to the different methods based on the query parameters that are present
func (h WithSSBHandler) decideMethod(w http.ResponseWriter, req *http.Request) {
	queryVals := req.URL.Query()

	var (
		alias string = queryVals.Get("alias")
		cid   *refs.FeedRef
	)

	if cidString := queryVals.Get("cid"); cidString != "" {
		parsedCID, err := refs.ParseFeedRef(cidString)
		if err == nil {
			cid = parsedCID
		}
	} else {
		aliasEntry, err := h.aliasesdb.Resolve(req.Context(), alias)
		if err == nil {
			cid = &aliasEntry.Feed
		}
	}

	// input either an alias or a feed reference
	// it is set by the landing form if non of the params are present
	if input := queryVals.Get("input"); input != "" {
		// assume ssb id first
		var err error
		cid, err = refs.ParseFeedRef(input)
		if err != nil {
			// try input as an alias
			aliasEntry, err := h.aliasesdb.Resolve(req.Context(), input)
			if err != nil {
				h.render.Error(w, req, http.StatusBadRequest, err)
				return
			}
			cid = &aliasEntry.Feed
			alias = aliasEntry.Name
		}

		// update cid for server-initiated
		queryVals.Set("cid", cid.Ref())
	}

	// ?cid=CID&cc=CC does client-initiated http-auth
	if cc := queryVals.Get("cc"); cc != "" && cid != nil {
		err := h.clientInitiated(w, req, *cid)
		if err != nil {
			h.render.Error(w, req, http.StatusInternalServerError, err)
		}
		return
	}

	//  without any query params: shows a form field so you can input alias or SSB ID
	if alias == "" && cid == nil {
		h.render.StaticHTML("auth/start_login_form.tmpl").ServeHTTP(w, req)
		return
	}

	// ?cid=CID does server-initiated http-auth
	// ?alias=ALIAS does server-initiated http-auth
	h.render.HTML("auth/withssb_server_start.tmpl", h.serverInitiated).ServeHTTP(w, req)
}

// clientInitiated is called with a client challange (?cc=123) and calls back to the passed client using muxrpc to request a signed solution
// if everything checks out it redirects to the admin dashboard
func (h WithSSBHandler) clientInitiated(w http.ResponseWriter, req *http.Request, client refs.FeedRef) error {
	queryParams := req.URL.Query()

	var clientReq signinwithssb.ClientRequest
	clientReq.ServerID = h.roomID // fill in the server

	// validate and update client challenge
	cc := queryParams.Get("cc")
	clientReq.ClientChallenge = cc

	// check that we have that member
	member, err := h.membersdb.GetByFeed(req.Context(), client)
	if err != nil {
		errMsg := fmt.Errorf("ssb http auth: client isn't a member: %w", err)
		if err == roomdb.ErrNotFound {
			return weberrors.ErrForbidden{Details: errMsg}
		}
		return errMsg
	}
	clientReq.ClientID = client

	// get the connected client for that member
	edp, connected := h.endpoints.GetEndpointFor(client)
	if !connected {
		return weberrors.ErrForbidden{Details: fmt.Errorf("ssb http auth: client not connected to room")}
	}

	// roll a Challenge from the server
	sc := signinwithssb.GenerateChallenge()
	clientReq.ServerChallenge = sc

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

	if !clientReq.Validate(solutionBytes) {
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

func (h WithSSBHandler) serverInitiated(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	sc := h.bridge.RegisterSession()

	// prepare the ssb-uri
	// https://ssb-ngi-pointer.github.io/ssb-http-auth-spec/#list-of-new-ssb-uris
	var queryParams = make(url.Values)
	queryParams.Set("action", "start-http-auth")
	queryParams.Set("sid", h.roomID.Ref())
	queryParams.Set("sc", sc)

	var startAuthURI url.URL
	startAuthURI.Scheme = "ssb"
	startAuthURI.Opaque = "experimental"
	startAuthURI.RawQuery = queryParams.Encode()

	// generate a QR code with the token inside so that you can open it easily in a supporting mobile app
	qrCode, err := qrcode.New(startAuthURI.String(), qrcode.Medium)
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

	// template.URL signals the template engine that those aren't fishy and from a trusted source
	type templateData struct {
		SSBURI          template.URL
		QRCodeURI       template.URL
		ServerChallenge string
	}
	data := templateData{
		SSBURI:    template.URL(startAuthURI.String()),
		QRCodeURI: template.URL(qrURI),

		ServerChallenge: sc,
	}
	return data, nil
}

// finalizeCookie is called with a redirect from the js sse client if everything worked
func (h WithSSBHandler) finalizeCookie(w http.ResponseWriter, r *http.Request) {
	tok := r.URL.Query().Get("token")

	// check the token is correct
	if _, err := h.sessiondb.CheckToken(r.Context(), tok); err != nil {
		http.Error(w, "invalid session token", http.StatusInternalServerError)
		return
	}

	if err := h.saveCookie(w, r, tok); err != nil {
		http.Error(w, "failed to save cookie", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusTemporaryRedirect)
}

// eventSource is the server-side of our server-sent events (SSE) session
// https://html.spec.whatwg.org/multipage/server-sent-events.html
func (h WithSSBHandler) eventSource(w http.ResponseWriter, r *http.Request) {
	flusher, err := w.(http.Flusher)
	if !err {
		http.Error(w, "ssb http auth: server-initiated method needs streaming support", http.StatusInternalServerError)
		return
	}
	notifier, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "ssb http auth: cant notify about closed requests", http.StatusInternalServerError)
		return
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
		time.Sleep(3 * time.Minute)
		tick.Stop()
		sender.send("ping", "Warning: reached waiting time of 3 minutes.")
		flusher.Flush()
		logger.Log("event", "stopped")
	}()

	start := time.Now()
	flusher.Flush()

	// Push events to client
	notify := notifier.CloseNotify() // closes when the http request is closed
	for {
		select {

		case <-notify:
			logger.Log("event", "request closed")
			return

		case <-tick.C:
			sender.send("ping", fmt.Sprintf("Waiting for solution (session age: %s)", time.Since(start)))
			logger.Log("event", "sent ping")

		case update := <-evtCh:
			data := "challenge validation failed"
			event := "failed"

			if update.Worked {
				data = update.Token
				event = "success"
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
