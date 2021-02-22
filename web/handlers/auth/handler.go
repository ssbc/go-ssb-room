// SPDX-License-Identifier: MIT

package auth

import (
	"net/http"

	"github.com/go-kit/kit/log/level"
	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"
	"go.mindeco.de/logging"

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

	m.Get(router.AuthFallbackSignIn).HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		logger := logging.FromContext(req.Context())
		level.Info(logger).Log("event", "authorize request")
		a.Authorize(w, req)
	})

	m.Get(router.AuthFallbackSignOut).HandlerFunc(a.Logout)

	return m
}
