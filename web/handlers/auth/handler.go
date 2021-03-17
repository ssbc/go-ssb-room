// SPDX-License-Identifier: MIT

package auth

import (
	"net/http"

	"github.com/gorilla/csrf"
	"github.com/gorilla/mux"
	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	refs "go.mindeco.de/ssb-refs"
)

var HTMLTemplates = []string{
	"auth/fallback_sign_in.tmpl",
	"auth/withssb_sign_in.tmpl",
}

func Handler(
	m *mux.Router,
	r *render.Renderer,
	ah *auth.Handler,
	roomID refs.FeedRef,
	endpoints network.Endpoints,
	aliasDB roomdb.AliasesService,
	membersDB roomdb.MembersService,
) http.Handler {
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

	var ssb withssbHandler
	ssb.roomID = roomID
	ssb.aliases = aliasDB
	ssb.members = membersDB
	ssb.endpoints = endpoints
	ssb.cookieAuth = ah
	m.Get(router.AuthWithSSBSignIn).HandlerFunc(r.HTML("auth/withssb_sign_in.tmpl", ssb.login))

	return m
}
