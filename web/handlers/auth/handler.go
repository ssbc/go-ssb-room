// SPDX-License-Identifier: MIT

package auth

import (
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

var HTMLTemplates = []string{
	"auth/fallback_sign_in.tmpl",
	"auth/withssb_sign_in.tmpl",
}

func NewFallbackPasswordHandler(
	m *mux.Router,
	r *render.Renderer,
	ah *auth.Handler,
) {
	if m == nil {
		m = router.Auth(nil)
	}

	// just the form
	m.Get(router.AuthFallbackSignInForm).Handler(r.HTML("auth/fallback_sign_in.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		return map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(req),
		}, nil
	}))

	// hook up the auth handler to the router
	m.Get(router.AuthFallbackSignIn).HandlerFunc(ah.Authorize)

	m.Get(router.AuthSignOut).HandlerFunc(ah.Logout)
}
