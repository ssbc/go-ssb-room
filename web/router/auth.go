// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AuthFallbackSignInForm = "auth:fallback:signin:form"
	AuthFallbackSignIn     = "auth:fallback:signin"
	AuthFallbackSignOut    = "auth:fallback:logout"

	AuthWithSSBSignIn  = "auth:ssb:signin"
	AuthWithSSBSignOut = "auth:ssb:logout"
)

// NewSignin constructs a mux.Router containing the routes for sign-in and -out
func Auth(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	// register fallback
	m.Path("/fallback/signin").Methods("GET").Name(AuthFallbackSignInForm)
	m.Path("/fallback/signin").Methods("POST").Name(AuthFallbackSignIn)
	m.Path("/fallback/logout").Methods("GET").Name(AuthFallbackSignOut)

	m.Path("/withssb/signin").Methods("GET").Name(AuthWithSSBSignIn)
	m.Path("/withssb/logout").Methods("GET").Name(AuthWithSSBSignOut)

	return m
}
