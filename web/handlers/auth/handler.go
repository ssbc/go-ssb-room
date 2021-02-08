package auth

import (
	"net/http"

	"github.com/gorilla/mux"
	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
)

var HTMLTemplates = []string{
	"/auth/fallback_sign_in.tmpl",
}

func Handler(m *mux.Router, r *render.Renderer, a *auth.Handler) http.Handler {
	if m == nil {
		m = router.Auth(nil)
	}

	m.Get(router.AuthFallbackSignInForm).Handler(r.StaticHTML("/auth/fallback_sign_in.tmpl"))
	m.Get(router.AuthFallbackSignIn).HandlerFunc(a.Authorize)
	m.Get(router.AuthFallbackSignOut).HandlerFunc(a.Logout)

	return m
}
