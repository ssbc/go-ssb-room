package admin

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
)

var HTMLTemplates = []string{
	"/admin/dashboard.tmpl",
}

func Handler(m *mux.Router, r *render.Renderer) http.Handler {
	if m == nil {
		m = router.Admin(nil)
	}

	m.Get(router.AdminDashboard).HandlerFunc(r.HTML("/admin/dashboard.tmpl", dashboard))

	return m
}

func dashboard(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	return struct {
		Name string
	}{"test"}, nil
}
