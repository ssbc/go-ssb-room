package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/gossb-rooms/internal/repo"
	"github.com/ssb-ngi-pointer/gossb-rooms/web"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/handlers/news"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/i18n"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
)

// New initializes the whole web stack for rooms, with all the sub-modules and routing.
func New(m *mux.Router, repo repo.Interface) (http.Handler, error) {
	if m == nil {
		m = router.CompleteApp()
	}

	locHelper, err := i18n.New(repo)
	if err != nil {
		return nil, err
	}

	r, err := render.New(web.Templates,
		render.BaseTemplates("/base.tmpl"),
		render.AddTemplates(append(news.HTMLTemplates,
			"/landing/index.tmpl",
			"/landing/about.tmpl",
			"/error.tmpl")...),
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

	// hookup handlers to the router
	m.PathPrefix("/news").Handler(http.StripPrefix("/news", news.Handler(m, r)))

	m.Get(router.CompleteIndex).Handler(r.StaticHTML("/landing/index.tmpl"))
	m.Get(router.CompleteAbout).Handler(r.StaticHTML("/landing/about.tmpl"))

	m.NotFoundHandler = http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		fmt.Fprintf(rw, "404: url not found")
	})

	// TODO: disable in non-dev
	return r.GetReloader()(m), nil
}
