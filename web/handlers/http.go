package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/repo"
	"github.com/ssb-ngi-pointer/gossb-rooms/web"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/handlers/admin"
	roomsAuth "github.com/ssb-ngi-pointer/gossb-rooms/web/handlers/auth"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/handlers/news"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/i18n"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
)

// New initializes the whole web stack for rooms, with all the sub-modules and routing.
func New(
	m *mux.Router,
	repo repo.Interface,
	as admindb.AuthWithSSBService,
	fs admindb.AuthFallbackService,
) (http.Handler, error) {
	if m == nil {
		m = router.CompleteApp()
	}

	locHelper, err := i18n.New(repo)
	if err != nil {
		return nil, err
	}

	r, err := render.New(web.Templates,
		render.BaseTemplates("/base.tmpl"),
		render.AddTemplates(concatTemplates(
			[]string{
				"/landing/index.tmpl",
				"/landing/about.tmpl",
				"/error.tmpl",
			},
			news.HTMLTemplates,
			roomsAuth.HTMLTemplates,
			admin.HTMLTemplates,
		)...),
		render.FuncMap(web.TemplateFuncs(m)),
		// TODO: add plural and template data variants
		// TODO: move these to the i18n helper pkg
		render.InjectTemplateFunc("i18n", func(r *http.Request) interface{} {
			lang := r.FormValue("lang")
			accept := r.Header.Get("Accept-Language")
			loc := locHelper.NewLocalizer(lang, accept)
			return loc.LocalizeSimple
		}),
	)
	if err != nil {
		return nil, fmt.Errorf("web Handler: failed to create renderer: %w", err)
	}

	cookieCodec, err := web.LoadOrCreateCookieSecrets(repo)
	if err != nil {
		return nil, err
	}

	store := &sessions.CookieStore{
		Codecs: cookieCodec,
		Options: &sessions.Options{
			Path:   "/",
			MaxAge: 30,
		},
	}

	// TODO: use r.Error?
	notAuthorizedH := r.HTML("/error.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		statusCode := http.StatusUnauthorized
		rw.WriteHeader(statusCode)
		return errorTemplateData{
			statusCode,
			"Unauthorized",
			"you are not authorized to access the requested site",
		}, nil
	})

	a, err := auth.NewHandler(fs,
		auth.SetStore(store),
		auth.SetNotAuthorizedHandler(notAuthorizedH),
	)
	if err != nil {
		return nil, fmt.Errorf("web Handler: failed to init fallback auth system: %w", err)
	}

	// hookup handlers to the router
	roomsAuth.Handler(m, r, a)

	adminRouter := m.PathPrefix("/admin").Subrouter()
	adminRouter.Use(a.Authenticate)

	// we dont strip path here because it somehow fucks with the middleware setup
	adminRouter.PathPrefix("/").Handler(admin.Handler(adminRouter, r))

	m.PathPrefix("/news").Handler(http.StripPrefix("/news", news.Handler(m, r)))

	m.Get(router.CompleteIndex).Handler(r.StaticHTML("/landing/index.tmpl"))
	m.Get(router.CompleteAbout).Handler(r.StaticHTML("/landing/about.tmpl"))

	m.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(web.Assets)))

	m.NotFoundHandler = r.HTML("/error.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		rw.WriteHeader(http.StatusNotFound)
		return errorTemplateData{http.StatusNotFound, "Not Found", "the requested page wasnt found.."}, nil
	})

	if web.Production {
		return m, nil
	}

	return r.GetReloader()(m), nil
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
