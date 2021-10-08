// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/russross/blackfriday/v2"
	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"
	"go.mindeco.de/log/level"
	"go.mindeco.de/logging"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomstate"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	weberrs "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/handlers/admin"
	roomsAuth "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/handlers/auth"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
)

var HTMLTemplates = []string{
	"landing/index.tmpl",
	"alias.tmpl",

	"change-member-password.tmpl",

	"invite/consumed.tmpl",
	"invite/facade.tmpl",
	"invite/facade-fallback.tmpl",
	"invite/insert-id.tmpl",

	"notice/list.tmpl",
	"notice/show.tmpl",

	"error.tmpl",
}

// Databases is an options stuct for the required databases of the web handlers
type Databases struct {
	Aliases       roomdb.AliasesService
	AuthFallback  roomdb.AuthFallbackService
	AuthWithSSB   roomdb.AuthWithSSBService
	Config        roomdb.RoomConfig
	DeniedKeys    roomdb.DeniedKeysService
	Invites       roomdb.InvitesService
	Notices       roomdb.NoticesService
	Members       roomdb.MembersService
	PinnedNotices roomdb.PinnedNoticesService
}

// New initializes the whole web stack for rooms, with all the sub-modules and routing.
func New(
	logger logging.Interface,
	repo repo.Interface,
	netInfo network.ServerEndpointDetails,
	roomState *roomstate.Manager,
	roomEndpoints network.Endpoints,
	bridge *signinwithssb.SignalBridge,
	dbs Databases,
) (http.Handler, error) {
	m := router.CompleteApp()
	urlTo := web.NewURLTo(m, netInfo)

	locHelper, err := i18n.New(repo, dbs.Config)
	if err != nil {
		return nil, err
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

	flashHelper := weberrs.NewFlashHelper(cookieStore, locHelper)

	eh := weberrs.NewErrorHandler(locHelper, flashHelper)

	allTheTemplates := concatTemplates(
		HTMLTemplates,
		roomsAuth.HTMLTemplates,
		admin.HTMLTemplates,
	)

	renderOpts := []render.Option{
		render.SetLogger(logger),
		render.BaseTemplates("base.tmpl", "menu.tmpl", "flashes.tmpl"),
		render.AddTemplates(allTheTemplates...),
		render.SetErrorHandler(eh.Handle),
		render.FuncMap(web.TemplateFuncs(m, netInfo)),

		render.InjectTemplateFunc("privacy_mode_is", func(r *http.Request) interface{} {
			return func(want string) bool {
				has, err := dbs.Config.GetPrivacyMode(r.Context())
				if err != nil {
					return false
				}
				return has.String() == want
			}
		}),

		render.InjectTemplateFunc("current_page_is", func(r *http.Request) interface{} {
			return func(routeName string) bool {
				route := m.Get(routeName)
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

		render.InjectTemplateFunc("language_count", func(r *http.Request) interface{} {
			return func() int {
				return len(locHelper.ListLanguages())
			}
		}),

		render.InjectTemplateFunc("list_languages", func(r *http.Request) interface{} {
			return func(postRoute *url.URL, classList string) (template.HTML, error) {
				languages := locHelper.ListLanguages()
				var buf bytes.Buffer

				for _, entry := range languages {
					data := changeLanguageTemplateData{
						PostRoute:    postRoute.String(),
						CSRFElement:  csrf.TemplateField(r),
						LangTag:      entry.Tag,
						RedirectPage: r.RequestURI,
						Translation:  entry.Translation,
						ClassList:    classList,
					}
					err = changeLanguageTemplate.Execute(&buf, data)
					if err != nil {
						return "", fmt.Errorf("Error while executing change language template: %w", err)
					}
				}

				return (template.HTML)(buf.String()), nil
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
				return urlTo(router.CompleteNoticeShow, "id", notice.ID)
			}
		}),
	}

	renderOpts = append(renderOpts, locHelper.GetRenderFuncs()...)
	renderOpts = append(renderOpts, members.TemplateHelpers(dbs.Config)...)

	r, err := render.New(web.Templates, renderOpts...)
	if err != nil {
		return nil, fmt.Errorf("web Handler: failed to create renderer: %w", err)
	}
	eh.SetRenderer(r)

	authWithPassword, err := auth.NewHandler(dbs.AuthFallback,
		auth.SetStore(cookieStore),
		auth.SetErrorHandler(func(rw http.ResponseWriter, req *http.Request, err error, code int) {
			eh.Handle(rw, req, code, err)
		}),
		auth.SetNotAuthorizedHandler(http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
			eh.Handle(rw, req, http.StatusForbidden, weberrs.ErrNotAuthorized)
		})),
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
		csrf.Path("/"),
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
		netInfo,
		roomEndpoints,
		dbs.Aliases,
		dbs.Members,
		dbs.AuthWithSSB,
		cookieStore,
		bridge,
	)

	// auth routes
	m.Get(router.AuthLogin).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		if label := req.URL.Query().Get("ssb-http-auth"); label != "" {
			authWithSSB.DecideMethod(w, req)
		} else {
			r.Render(w, req, "auth/decide_method.tmpl", http.StatusOK, nil)
		}
	})

	m.Get(router.AuthFallbackFinalize).HandlerFunc(authWithPassword.Authorize)
	m.Get(router.AuthFallbackLogin).Handler(r.HTML("auth/fallback_sign_in.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		pageData := map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(req),
		}
		pageData["Flashes"], err = flashHelper.GetAll(w, req)
		if err != nil {
			return nil, err
		}
		return pageData, nil
	}))
	m.Get(router.AuthLogout).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		err = authWithSSB.Logout(w, req)
		if err != nil {
			level.Warn(logging.FromContext(req.Context())).Log("err", err)
		}
		authWithPassword.Logout(w, req)
	})

	// all the admin routes
	adminHandler := admin.Handler(
		netInfo,
		r,
		roomState,
		flashHelper,
		locHelper,
		admin.Databases{
			Aliases:       dbs.Aliases,
			AuthFallback:  dbs.AuthFallback,
			Config:        dbs.Config,
			DeniedKeys:    dbs.DeniedKeys,
			Invites:       dbs.Invites,
			Notices:       dbs.Notices,
			Members:       dbs.Members,
			PinnedNotices: dbs.PinnedNotices,
		},
	)
	mainMux.Handle("/admin/", members.AuthenticateFromContext(r)(adminHandler))

	var mh = newMembersHandler(netInfo.Development, r, urlTo, flashHelper, dbs.AuthFallback)
	m.Get(router.MembersChangePasswordForm).HandlerFunc(r.HTML("change-member-password.tmpl", mh.changePasswordForm))
	m.Get(router.MembersChangePassword).HandlerFunc(mh.changePassword)

	// handle setting language
	m.Get(router.CompleteSetLanguage).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		lang := req.FormValue("lang")
		previousRoute := req.FormValue("page")

		session, err := cookieStore.Get(req, i18n.LanguageCookieName)
		if err != nil {
			eh.Handle(w, req, http.StatusInternalServerError, err)
			return
		}

		session.Values["lang"] = lang
		err = session.Save(req, w)
		if err != nil {
			err = fmt.Errorf("we failed to save the language session cookie %w\n", err)
			eh.Handle(w, req, http.StatusInternalServerError, err)
			return
		}

		http.Redirect(w, req, previousRoute, http.StatusSeeOther)
	})

	// landing page
	m.Get(router.CompleteIndex).Handler(r.HTML("landing/index.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		// TODO: try websocket upgrade (issue #)

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

	// notices (the mini-CMS)
	var nh = noticeHandler{
		flashes: flashHelper,

		notices: dbs.Notices,
		pinned:  dbs.PinnedNotices,
	}
	m.Get(router.CompleteNoticeList).Handler(r.HTML("notice/list.tmpl", nh.list))
	m.Get(router.CompleteNoticeShow).Handler(r.HTML("notice/show.tmpl", nh.show))

	// public aliases
	var ah = aliasHandler{
		r: r,

		db:     dbs.Aliases,
		config: dbs.Config,

		roomEndpoint: netInfo,
	}
	m.Get(router.CompleteAliasResolve).HandlerFunc(ah.resolve)

	//public invites
	var ih = inviteHandler{
		render:      r,
		urlTo:       urlTo,
		networkInfo: netInfo,

		config:        dbs.Config,
		pinnedNotices: dbs.PinnedNotices,
		invites:       dbs.Invites,
		deniedKeys:    dbs.DeniedKeys,
	}
	m.Get(router.CompleteInviteFacade).HandlerFunc(ih.presentFacade)
	m.Get(router.CompleteInviteFacadeFallback).Handler(r.HTML("invite/facade-fallback.tmpl", ih.presentFacadeFallback))
	m.Get(router.CompleteInviteInsertID).Handler(r.HTML("invite/insert-id.tmpl", ih.presentInsert))
	m.Get(router.CompleteInviteConsume).HandlerFunc(ih.consume)
	m.Get(router.OpenModeCreateInvite).HandlerFunc(r.HTML("admin/invite-created.tmpl", ih.createOpenMode))

	// static assets
	m.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(web.Assets)))

	// TODO: doesnt work because of of mainMux wrapper, see issue #35
	m.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		eh.Handle(rw, req, http.StatusNotFound, weberrs.PageNotFound{Path: req.URL.Path})
	})

	// hook up main stdlib mux to the gorrilla/mux with named routes
	// TODO: issue #35
	mainMux.Handle("/", m)

	consumeURL := urlTo(router.CompleteInviteConsume)

	// apply HTTP middleware
	middlewares := []func(http.Handler) http.Handler{
		members.ContextInjecter(dbs.Members, authWithPassword, authWithSSB),
		CSRF,

		// We disable CSRF for certain requests that are done by apps
		// only if they already contain some secret (like invite consumption)
		func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ct := req.Header.Get("Content-Type")
				if req.URL.Path == consumeURL.Path && ct == "application/json" {
					next.ServeHTTP(w, csrf.UnsafeSkipCheck(req))
					return
				}
				next.ServeHTTP(w, req)
			})
		},

		logging.InjectHandler(logger),
		logging.RecoveryHandler(),
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
func concatTemplates(lst ...[]string) []string {
	var catted []string

	for _, tpls := range lst {
		for _, t := range tpls {
			catted = append(catted, t)
		}

	}
	return catted
}
