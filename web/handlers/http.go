package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"go.mindeco.de/http/render"

	"go.mindeco.de/ssb-rooms/web"
	"go.mindeco.de/ssb-rooms/web/handlers/news"
	"go.mindeco.de/ssb-rooms/web/router"
)

func New(m *mux.Router) (http.Handler, error) {
	if m == nil {
		m = router.CompleteApp()
	}

	r, err := render.New(web.Assets,
		render.BaseTemplates("/base.tmpl"),
		render.AddTemplates(append(news.HTMLTemplates,
			"/landing/index.tmpl",
			"/landing/about.tmpl",
			"/error.tmpl")...),
		render.FuncMap(web.TemplateFuncs(m)),
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
	return m, nil
}
