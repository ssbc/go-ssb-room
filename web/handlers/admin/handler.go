// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

var HTMLTemplates = []string{
	"/admin/dashboard.tmpl",
}

func Handler(m *mux.Router, r *render.Renderer, roomState *roomstate.Manager) http.Handler {
	if m == nil {
		m = router.Admin(nil)
	}

	m.Get(router.AdminDashboard).HandlerFunc(r.HTML("/admin/dashboard.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		lst := roomState.List()
		return struct {
			Clients []string
			Count   int
		}{lst, len(lst)}, nil
	}))

	return m
}
