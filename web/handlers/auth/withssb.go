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

// WithSSBHandler implements the oauth-like challenge/response dance described in
// https://ssb-ngi-pointer.github.io/rooms2/#sign-in-with-ssb
type WithSSBHandler struct {
	roomID refs.FeedRef

	membersdb roomdb.MembersService
	aliasesdb roomdb.AliasesService
	sessiondb roomdb.AuthWithSSBService

	cookieStore sessions.Store

	endpoints network.Endpoints

	bridge *signinwithssb.SignalBridge
}

type registerToEventSourceMap map[string]<-chan signinwithssb.Event

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
	ssb.roomID = roomID
	ssb.aliasesdb = aliasDB
	ssb.membersdb = membersDB
	ssb.endpoints = endpoints
	ssb.sessiondb = sessiondb
	ssb.cookieStore = cookies
	ssb.bridge = bridge

	m.Get(router.AuthWithSSBSignIn).HandlerFunc(r.HTML("auth/withssb_sign_in.tmpl", ssb.login))

	m.HandleFunc("/sse/login/{sc}", r.HTML("auth/withssb_server_start.tmpl", ssb.startWithServer))
	m.HandleFunc("/sse/events", ssb.eventSource)

	return &ssb
}

func (h WithSSBHandler) login(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	queryParams := req.URL.Query()

	var clientReq signinwithssb.ClientRequest
	clientReq.ServerID = h.roomID // fill in the server

	// validate and update client challenge
	cc := queryParams.Get("cc")
	if _, err := signinwithssb.DecodeChallengeString(cc); err != nil {
		return nil, weberrors.ErrBadRequest{Where: "client-challenge", Details: err}
	}
	clientReq.ClientChallenge = cc

	// check who the client is
	var client refs.FeedRef
	if cid := queryParams.Get("cid"); cid != "" {
		parsed, err := refs.ParseFeedRef(cid)
		if err != nil {
			return nil, weberrors.ErrBadRequest{Where: "cid", Details: err}
		}
		client = *parsed
	} else {
		alias, err := h.aliasesdb.Resolve(req.Context(), queryParams.Get("alias"))
		if err != nil {
			return nil, weberrors.ErrBadRequest{Where: "alias", Details: err}
		}
		client = alias.Feed
	}

	// check that we have that member
	member, err := h.membersdb.GetByFeed(req.Context(), client)
	if err != nil {
		if err == roomdb.ErrNotFound {
			return nil, weberrors.ErrForbidden{Details: fmt.Errorf("sign-in: client isnt a member")}
		}
		return nil, err
	}
	clientReq.ClientID = client

	// get the connected client for that member
	edp, connected := h.endpoints.GetEndpointFor(client)
	if !connected {
		return nil, weberrors.ErrForbidden{Details: fmt.Errorf("sign-in: client not connected to room")}
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
		return nil, err
	}

	// decode and validate the response
	solutionBytes, err := base64.URLEncoding.DecodeString(solution)
	if err != nil {
		return nil, err
	}

	if !clientReq.Validate(solutionBytes) {
		return nil, fmt.Errorf("sign-in with ssb: validation of client solution failed")
	}

	// create a session for invalidation
	tok, err := h.sessiondb.CreateToken(req.Context(), member.ID)
	if err != nil {
		return nil, err
	}

	session, err := h.cookieStore.Get(req, siwssbSessionName)
	if err != nil {
		return nil, err
	}

	session.Values[memberToken] = tok
	session.Values[userTimeout] = time.Now().Add(lifetime)
	if err := session.Save(req, w); err != nil {
		return nil, err
	}

	return "you are now logged in!", nil
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

const lifetime = time.Hour * 24

// Authenticate calls the next unless AuthenticateRequest returns an error
func (h WithSSBHandler) Authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := h.AuthenticateRequest(r); err != nil {
			// TODO: render.Error
			http.Error(w, weberrors.ErrNotAuthorized.Error(), http.StatusForbidden)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// AuthenticateRequest uses the passed request to load and return the session data that was stored previously.
// If it is invalid or there is no session, it will return ErrNotAuthorized.
// Otherwise it will return the member ID that belongs to the session.
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
func (h WithSSBHandler) Logout(w http.ResponseWriter, r *http.Request) {
	session, err := h.cookieStore.Get(r, siwssbSessionName)
	if err != nil {
		// TODO: render.Error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// ah.errorHandler(w, r, err, http.StatusInternalServerError)
		return
	}

	session.Values[userTimeout] = time.Now().Add(-lifetime)
	session.Options.MaxAge = -1
	if err := session.Save(r, w); err != nil {
		// TODO: render.Error
		http.Error(w, err.Error(), http.StatusInternalServerError)
		// ah.errorHandler(w, r, err, http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/", http.StatusSeeOther)
	return
}

// server-sent-events stuff

func (h WithSSBHandler) startWithServer(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	sc := h.bridge.RegisterSession()

	var queryParams = make(url.Values)
	queryParams.Set("action", "start-http-auth")

	var startAuthURI url.URL
	startAuthURI.Scheme = "ssb"
	startAuthURI.Opaque = "experimental"
	startAuthURI.RawQuery = queryParams.Encode()

	qrCode, err := qrcode.New(startAuthURI.String(), qrcode.High)
	if err != nil {
		return nil, err
	}
	qrCode.BackgroundColor = color.RGBA{R: 0xf9, G: 0xfa, B: 0xfb}
	qrCode.ForegroundColor = color.Black

	qrCodeData, err := qrCode.PNG(-8)
	if err != nil {
		return nil, err
	}

	qrURI := "data:image/png;base64," + base64.StdEncoding.EncodeToString(qrCodeData)

	return struct {
		SSBURI          template.URL
		QRCodeURI       template.URL
		ServerChallenge string
	}{template.URL(startAuthURI.String()), template.URL(qrURI), sc}, nil
}

type event struct {
	ID    uint32
	Data  string
	Event string
}

func (h WithSSBHandler) eventSource(w http.ResponseWriter, r *http.Request) {
	flusher, err := w.(http.Flusher)
	if !err {
		http.Error(w, "Streaming unsupported!", http.StatusInternalServerError)
		return
	}
	notifier, ok := w.(http.CloseNotifier)
	if !ok {
		http.Error(w, "cant notify about closed requests", http.StatusInternalServerError)
		return
	}

	// setup server-sent events
	// https://html.spec.whatwg.org/multipage/server-sent-events.html
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Transfer-Encoding", "chunked")

	// Get the StreamID from the URL
	sc := r.URL.Query().Get("sc")
	if sc == "" {
		http.Error(w, "Please specify a stream!", http.StatusBadRequest)
		return
	}

	logger := logging.FromContext(r.Context())
	logger = level.Debug(logger)
	logger = kitlog.With(logger, "stream", sc)
	logger.Log("event", "stream opened")

	evtCh, has := h.bridge.GetEventChannel(sc)
	if !has {
		http.Error(w, "No such session!", http.StatusBadRequest)
		return
	}

	tick := time.NewTicker(3 * time.Second)
	go func() {
		time.Sleep(3 * time.Minute)
		tick.Stop()
		logger.Log("event", "stopped")
	}()

	start := time.Now()
	flusher.Flush()

	// Push events to client
	var (
		evtID = uint32(0)
	)

	notify := notifier.CloseNotify()
	for {
		select {

		case <-notify:
			logger.Log("event", "request closed")
			return

		case <-tick.C:
			msg := fmt.Sprintf("Waiting for solution (session age: %s)", time.Since(start))
			sendServerEvent(w, event{
				ID:    evtID,
				Data:  msg,
				Event: "ping",
			})
			logger.Log("event", "sent", "ping", evtID)

		case update := <-evtCh:
			evt := event{
				ID:    evtID,
				Data:  "challange validation failed",
				Event: "failed",
			}

			if update.Worked {
				evt.Event = "success"
				evt.Data = update.Token
			}

			sendServerEvent(w, evt)

			logger.Log("event", "sent", "worked", update.Worked)
		}
		evtID++
		flusher.Flush()
	}
}

func sendServerEvent(w io.Writer, evt event) {
	fmt.Fprintf(w, "id: %d\n", evt.ID)
	fmt.Fprintf(w, "data: %s\n", evt.Data)
	if len(evt.Event) > 0 {
		fmt.Fprintf(w, "event: %s\n", evt.Event)
	}
	fmt.Fprint(w, "\n")
}
