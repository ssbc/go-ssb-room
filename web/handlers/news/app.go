package news

import (
	"net/http"

	"github.com/gorilla/mux"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
	"go.mindeco.de/http/render"
)

var HTMLTemplates = []string{
	"/news/overview.tmpl",
	"/news/post.tmpl",
}

// Handler creates a http.Handler with all the archives routes attached to it
func Handler(m *mux.Router, r *render.Renderer) http.Handler {
	if m == nil {
		m = router.News(nil)
	}

	m.Get(router.NewsOverview).Handler(r.HTML("/news/overview.tmpl", showOverview))
	m.Get(router.NewsPost).Handler(r.HTML("/news/post.tmpl", showPost))

	return m
}
