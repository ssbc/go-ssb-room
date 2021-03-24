// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AuthFallbackSignInForm = "auth:fallback:signin:form"
	AuthFallbackSignIn     = "auth:fallback:signin"

	AuthWithSSBSignIn = "auth:ssb:login"
	// AuthWithSSBSignIn

	AuthSignOut = "auth:logout"
)

// Auth constructs a mux.Router containing the routes for sign-in and -out
func Auth(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	// register password fallback
	m.Path("/password/signin").Methods("GET").Name(AuthFallbackSignInForm)
	m.Path("/password/signin").Methods("POST").Name(AuthFallbackSignIn)

	m.Path("/login").Methods("GET").Name(AuthWithSSBSignIn)
	m.Path("/logout").Methods("GET").Name(AuthSignOut)

	return m
}
