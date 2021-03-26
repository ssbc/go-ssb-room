// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AuthFallbackSignIn = "auth:fallback:signin"

	AuthLogin  = "auth:login"
	AuthLogout = "auth:logout"

	AuthWithSSBServerEvents = "auth:withssb:sse"
	AuthWithSSBFinalize     = "auth:withssb:finalize"
)

// Auth constructs a mux.Router containing the routes for sign-in and -out
func Auth(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	m.Path("/login").Methods("GET").Name(AuthLogin)
	m.Path("/logout").Methods("GET").Name(AuthLogout)

	// register password fallback
	m.Path("/password/signin").Methods("POST").Name(AuthFallbackSignIn)

	m.Path("/withssb/events").Methods("GET").Name(AuthWithSSBServerEvents)
	m.Path("/withssb/finalize").Methods("GET").Name(AuthWithSSBFinalize)

	return m
}
