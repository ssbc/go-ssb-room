// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"

	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
)

var HTMLTemplates = []string{
	"admin/dashboard.tmpl",
	"admin/menu.tmpl",
	"admin/allow-list.tmpl",
	"admin/allow-list-remove-confirm.tmpl",
}

// Handler supplies the elevated access pages to known users.
// It is not registering on the mux router like other pages to clean up the authorize flow.
func Handler(r *render.Renderer, roomState *roomstate.Manager, al admindb.AllowListService) http.Handler {
	mux := &http.ServeMux{}

	mux.HandleFunc("/dashboard", r.HTML("admin/dashboard.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		lst := roomState.List()
		return struct {
			Clients []string
			Count   int
		}{lst, len(lst)}, nil
	}))
	mux.HandleFunc("/menu", r.HTML("admin/menu.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		return map[string]interface{}{}, nil
	}))

	var ah = allowListH{
		r:  r,
		al: al,
	}

	mux.HandleFunc("/members", r.HTML("admin/allow-list.tmpl", ah.overview))
	mux.HandleFunc("/members/add", ah.add)
	mux.HandleFunc("/members/remove/confirm", r.HTML("admin/allow-list-remove-confirm.tmpl", ah.removeConfirm))
	mux.HandleFunc("/members/remove", ah.remove)

	return customStripPrefix("/admin", mux)
}

// trim prefix if exists (workaround for named router problem)
func customStripPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, prefix)
		rp := strings.TrimPrefix(r.URL.RawPath, prefix)
		if len(p) < len(r.URL.Path) && (r.URL.RawPath == "" || len(rp) < len(r.URL.RawPath)) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			r2.URL.RawPath = rp
			h.ServeHTTP(w, r2)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}
