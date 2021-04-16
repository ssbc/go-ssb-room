// SPDX-License-Identifier: MIT

package handlers

import (
	"fmt"
	"html/template"
	"net/http"
	"net/url"
  "strings"
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
	weberrs "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/handlers/admin"
	roomsAuth "github.com/ssb-ngi-pointer/go-ssb-room/web/handlers/auth"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

var HTMLTemplates = []string{
	"landing/index.tmpl",
	"landing/about.tmpl",
	"alias.tmpl",

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
			csrfElement := csrf.TemplateField(r)

			createFormElement := func(postRoute, tag, translation, classList string) string {
				return fmt.Sprintf(`
            <form
              action="%s"
              method="POST"
              >
              %s
              <input type="hidden" name="lang" value="%s">
              <input type="hidden" name="page" value="%s">
              <input
                type="submit"
                value="%s"
                class="%s"
                />
            </form>
            `, postRoute, csrfElement, tag, r.RequestURI, translation, classList)
			}
			return func(postRoute *url.URL, classList string) template.HTML {
				languages := locHelper.ListLanguages()
				languageOptions := make([]string, len(languages))
				for tag, translation := range languages {
					languageOptions = append(languageOptions, createFormElement(postRoute.String(), tag, translation, classList))
				}
				return (template.HTML)(strings.Join(languageOptions, "\n"))
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
	renderOpts = append(renderOpts, members.TemplateHelpers()...)

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
			Config:        dbs.Config,
			DeniedKeys:    dbs.DeniedKeys,
			Invites:       dbs.Invites,
			Notices:       dbs.Notices,
			Members:       dbs.Members,
			PinnedNotices: dbs.PinnedNotices,
		},
	)
	mainMux.Handle("/admin/", members.AuthenticateFromContext(r)(adminHandler))

	// handle setting language
	m.Get(router.CompleteSetLanguage).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		lang := req.FormValue("lang")
		previousRoute := req.FormValue("page")

		session, err := cookieStore.Get(req, i18n.LanguageCookieName)
		if err != nil {
			fmt.Errorf("cookie error? %w\n", err)
			return
		}

		session.Values["lang"] = lang
		err = session.Save(req, w)
		if err != nil {
			fmt.Errorf("we failed to save the language session cookie %w\n", err)
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
	m.Get(router.CompleteAbout).Handler(r.StaticHTML("landing/about.tmpl"))

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
	m.Get(router.CompleteInviteFacade).Handler(r.HTML("invite/facade.tmpl", ih.presentFacade))
	m.Get(router.CompleteInviteFacadeFallback).Handler(r.HTML("invite/facade-fallback.tmpl", ih.presentFacadeFallback))
	m.Get(router.CompleteInviteInsertID).Handler(r.HTML("invite/insert-id.tmpl", ih.presentInsert))
	m.Get(router.CompleteInviteConsume).HandlerFunc(ih.consume)

	// statuc assets
	m.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(web.Assets)))

	m.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		eh.Handle(rw, req, http.StatusNotFound, weberrs.PageNotFound{Path: req.URL.Path})
	})

	// hook up main stdlib mux to the gorrilla/mux with named routes
	mainMux.Handle("/", m)

	consumeURL := urlTo(router.CompleteInviteConsume)

	// apply HTTP middleware
	middlewares := []func(http.Handler) http.Handler{
		logging.RecoveryHandler(),
		logging.InjectHandler(logger),
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
