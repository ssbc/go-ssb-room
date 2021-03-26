// SPDX-License-Identifier: MIT

package handlers

import (
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/russross/blackfriday/v2"
	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"
	"go.mindeco.de/logging"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/handlers/admin"
	roomsAuth "github.com/ssb-ngi-pointer/go-ssb-room/web/handlers/auth"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	refs "go.mindeco.de/ssb-refs"
)

var HTMLTemplates = []string{
	"landing/index.tmpl",
	"landing/about.tmpl",
	"aliases-resolved.html",
	"invite/accept.tmpl",
	"invite/consumed.tmpl",
	"auth/fallback_sign_in.tmpl",
	"notice/list.tmpl",
	"notice/show.tmpl",
	"error.tmpl",
}

// Databases is an options stuct for the required databases of the web handlers
type Databases struct {
	Aliases       roomdb.AliasesService
	AuthFallback  roomdb.AuthFallbackService
	AuthWithSSB   roomdb.AuthWithSSBService
	DeniedKeys    roomdb.DeniedKeysService
	Invites       roomdb.InvitesService
	Notices       roomdb.NoticesService
	Members       roomdb.MembersService
	PinnedNotices roomdb.PinnedNoticesService
}

// NetworkInfo encapsulates the domain name of the room, it's ssb/secret-handshake public key and the HTTP and MUXRPC TCP ports.
type NetworkInfo struct {
	PortMUXRPC uint
	PortHTTPS  uint // 0 assumes default (443)

	RoomID refs.FeedRef

	Domain string
}

// New initializes the whole web stack for rooms, with all the sub-modules and routing.
func New(
	logger logging.Interface,
	repo repo.Interface,
	netInfo NetworkInfo,
	roomState *roomstate.Manager,
	roomEndpoints network.Endpoints,
	bridge *signinwithssb.SignalBridge,
	dbs Databases,
) (http.Handler, error) {
	m := router.CompleteApp()

	locHelper, err := i18n.New(repo)
	if err != nil {
		return nil, err
	}

	r, err := render.New(web.Templates,
		render.SetLogger(logger),
		render.BaseTemplates("base.tmpl", "menu.tmpl"),
		render.AddTemplates(concatTemplates(
			HTMLTemplates,
			roomsAuth.HTMLTemplates,
			admin.HTMLTemplates,
		)...),
		render.ErrorTemplate("error.tmpl"),
		render.FuncMap(web.TemplateFuncs(m)),
		// TODO: move these to the i18n helper pkg
		render.InjectTemplateFunc("i18npl", func(r *http.Request) interface{} {
			loc := i18n.LocalizerFromRequest(locHelper, r)
			return loc.LocalizePlurals
		}),
		render.InjectTemplateFunc("i18n", func(r *http.Request) interface{} {
			loc := i18n.LocalizerFromRequest(locHelper, r)
			return loc.LocalizeSimple
		}),
		render.InjectTemplateFunc("current_page_is", func(r *http.Request) interface{} {
			return func(routeName string) bool {
				route := router.CompleteApp().Get(routeName)
				if route == nil {
					return false
				}
				url, err := route.URLPath()
				if err != nil {
					return false
				}
				return r.RequestURI == url.Path
			}
		}),
		render.InjectTemplateFunc("urlToNotice", func(r *http.Request) interface{} {
			return func(name string) *url.URL {
				noticeName := roomdb.PinnedNoticeName(name)
				if !noticeName.Valid() {
					return nil
				}
				notice, err := dbs.PinnedNotices.Get(r.Context(), noticeName, "en-GB")
				if err != nil {
					return nil
				}
				route := router.CompleteApp().GetRoute(router.CompleteNoticeShow)
				if route == nil {
					return nil
				}
				u, err := route.URLPath()
				if err != nil {
					return nil
				}
				noticeID := strconv.FormatInt(notice.ID, 10)
				q := u.Query()
				q.Add("id", noticeID)
				u.RawQuery = q.Encode()
				return u
			}
		}),
		render.InjectTemplateFunc("is_logged_in", members.TemplateHelper()),
	)
	if err != nil {
		return nil, fmt.Errorf("web Handler: failed to create renderer: %w", err)
	}

	cookieCodec, err := web.LoadOrCreateCookieSecrets(repo)
	if err != nil {
		return nil, err
	}

	cookieStore := &sessions.CookieStore{
		Codecs: cookieCodec,
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 2 * 60 * 60, // two hours in seconds  // TODO: configure
		},
	}

	// TODO: this is just the error handler for http/auth, not render
	authErrH := func(rw http.ResponseWriter, req *http.Request, err error, code int) {
		var ih = i18n.LocalizerFromRequest(locHelper, req)

		// default, unlocalized message
		msg := err.Error()

		// localize some specific error messages
		var (
			aa roomdb.ErrAlreadyAdded
		)
		switch {
		case err == auth.ErrBadLogin:
			msg = ih.LocalizeSimple("AuthErrorBadLogin")

		case errors.Is(err, roomdb.ErrNotFound):
			msg = ih.LocalizeSimple("ErrorNotFound")

		case errors.As(err, &aa):
			msg = ih.LocalizeSimple("ErrorAlreadyAdded")
		}

		r.HTML("error.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
			return errorTemplateData{
				Err: msg,
				// TODO: localize?
				Status:     http.StatusText(code),
				StatusCode: code,
			}, nil
		}).ServeHTTP(rw, req)
	}

	notAuthorizedH := r.HTML("error.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		statusCode := http.StatusUnauthorized
		rw.WriteHeader(statusCode)
		return errorTemplateData{
			statusCode,
			"Unauthorized",
			"you are not authorized to access the requested site",
		}, nil
	})

	authWithPassword, err := auth.NewHandler(dbs.AuthFallback,
		auth.SetStore(cookieStore),
		auth.SetErrorHandler(authErrH),
		auth.SetNotAuthorizedHandler(notAuthorizedH),
		auth.SetLifetime(2*time.Hour), // TODO: configure
	)
	if err != nil {
		return nil, fmt.Errorf("web Handler: failed to init fallback auth system: %w", err)
	}

	// Cross Site Request Forgery prevention middleware
	csrfKey, err := web.LoadOrCreateCSRFSecret(repo)
	if err != nil {
		return nil, err
	}

	CSRF := csrf.Protect(csrfKey,
		csrf.ErrorHandler(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			err := csrf.FailureReason(req)
			// TODO: localize error?
			r.Error(w, req, http.StatusForbidden, err)
		})),
	)

	// this router is a bit of a qurik
	// TODO: explain problem between gorilla/mux named routers and authentication
	mainMux := &http.ServeMux{}

	// start hooking up handlers to the router

	authWithSSB := roomsAuth.NewWithSSBHandler(
		m,
		r,
		netInfo.RoomID,
		roomEndpoints,
		dbs.Aliases,
		dbs.Members,
		dbs.AuthWithSSB,
		cookieStore,
		bridge,
	)

	m.Get(router.AuthLogin).Handler(r.StaticHTML("auth/decide_method.tmpl"))

	m.Get(router.AuthFallbackFinalize).HandlerFunc(authWithPassword.Authorize)

	m.Get(router.AuthFallbackLogin).Handler(r.HTML("auth/fallback_sign_in.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		return map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(req),
		}, nil
	}))

	m.Get(router.AuthLogout).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err = authWithSSB.Logout(w, req)
		if err != nil {
			level.Warn(logging.FromContext(req.Context())).Log("err", err)
		}
		authWithPassword.Logout(w, req)
	})

	adminHandler := admin.Handler(
		netInfo.Domain,
		r,
		roomState,
		admin.Databases{
			Aliases:       dbs.Aliases,
			DeniedKeys:    dbs.DeniedKeys,
			Invites:       dbs.Invites,
			Notices:       dbs.Notices,
			Members:       dbs.Members,
			PinnedNotices: dbs.PinnedNotices,
		},
	)
	mainMux.Handle("/admin/", members.AuthenticateFromContext(r)(adminHandler))

	m.Get(router.CompleteIndex).Handler(r.HTML("landing/index.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		notice, err := dbs.PinnedNotices.Get(req.Context(), roomdb.NoticeDescription, "en-GB")
		if err != nil {
			return nil, fmt.Errorf("failed to find description: %w", err)
		}
		markdown := blackfriday.Run([]byte(notice.Content), blackfriday.WithNoExtensions())
		return noticeShowData{
			ID:       notice.ID,
			Title:    notice.Title,
			Content:  template.HTML(markdown),
			Language: notice.Language,
		}, nil
	}))
	m.Get(router.CompleteAbout).Handler(r.StaticHTML("landing/about.tmpl"))

	var nh = noticeHandler{
		notices: dbs.Notices,
		pinned:  dbs.PinnedNotices,
	}
	m.Get(router.CompleteNoticeList).Handler(r.HTML("notice/list.tmpl", nh.list))
	m.Get(router.CompleteNoticeShow).Handler(r.HTML("notice/show.tmpl", nh.show))

	var ah = aliasHandler{
		r: r,

		db: dbs.Aliases,

		roomID:            netInfo.RoomID,
		muxrpcHostAndPort: fmt.Sprintf("%s:%d", netInfo.Domain, netInfo.PortMUXRPC),
	}
	m.Get(router.CompleteAliasResolve).HandlerFunc(ah.resolve)

	var ih = inviteHandler{
		invites: dbs.Invites,

		roomPubKey:        netInfo.RoomID.PubKey(),
		muxrpcHostAndPort: fmt.Sprintf("%s:%d", netInfo.Domain, netInfo.PortMUXRPC),
	}
	m.Get(router.CompleteInviteAccept).Handler(r.HTML("invite/accept.tmpl", ih.acceptForm))
	m.Get(router.CompleteInviteConsume).Handler(r.HTML("invite/consumed.tmpl", ih.consume))

	m.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(web.Assets)))

	m.NotFoundHandler = r.HTML("error.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		rw.WriteHeader(http.StatusNotFound)
		msg := i18n.LocalizerFromRequest(locHelper, req).LocalizeSimple("PageNotFound")
		return errorTemplateData{http.StatusNotFound, "Not Found", msg}, nil
	})

	mainMux.Handle("/", m)

	// apply HTTP middleware
	middlewares := []func(http.Handler) http.Handler{
		logging.InjectHandler(logger),
		members.ContextInjecter(dbs.Members, authWithPassword, authWithSSB),
		CSRF,
	}

	if !web.Production {
		middlewares = append(middlewares, r.GetReloader())
	}

	var finalHandler http.Handler = mainMux
	for _, applyMiddleware := range middlewares {
		finalHandler = applyMiddleware(finalHandler)
	}

	return finalHandler, nil
}

// utils

type errorTemplateData struct {
	StatusCode int
	Status     string
	Err        string
}

func concatTemplates(lst ...[]string) []string {
	var catted []string

	for _, tpls := range lst {
		for _, t := range tpls {
			catted = append(catted, t)
		}

	}
	return catted
}
