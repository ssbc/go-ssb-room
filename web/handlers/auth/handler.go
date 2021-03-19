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
}

func Handler(m *mux.Router, r *render.Renderer, a *auth.Handler) http.Handler {
	if m == nil {
		m = router.Auth(nil)
	}

	m.Get(router.AuthFallbackSignInForm).Handler(r.HTML("auth/fallback_sign_in.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		return map[string]interface{}{
			csrf.TemplateTag: csrf.TemplateField(req),
		}, nil
	}))

	// hook up the auth handler to the router
	m.Get(router.AuthFallbackSignIn).HandlerFunc(ah.Authorize)
	m.Get(router.AuthFallbackSignOut).HandlerFunc(ah.Logout)

	return m
}
