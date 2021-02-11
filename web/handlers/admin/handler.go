// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
)

var HTMLTemplates = []string{
	"/admin/dashboard.tmpl",
}

// Handler supplies the elevated access pages to known users.
// It is not registering on the mux router like other pages to clean up the authorize flow.
func Handler(r *render.Renderer, roomState *roomstate.Manager) http.Handler {

	return r.HTML("/admin/dashboard.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		lst := roomState.List()
		return struct {
			Clients []string
			Count   int
		}{lst, len(lst)}, nil
	})
}
