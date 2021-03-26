// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AuthLogin  = "auth:login"
	AuthLogout = "auth:logout"

	AuthFallbackLogin    = "auth:fallback:login"
	AuthFallbackFinalize = "auth:fallback:finalize"

	AuthWithSSBLogin        = "auth:withssb:login"
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

	m.Path("/fallback/login").Methods("GET").Name(AuthFallbackLogin)
	m.Path("/fallback/finalize").Methods("POST").Name(AuthFallbackFinalize)

	m.Path("/withssb/login").Methods("GET").Name(AuthWithSSBLogin)
	m.Path("/withssb/events").Methods("GET").Name(AuthWithSSBServerEvents)
	m.Path("/withssb/finalize").Methods("GET").Name(AuthWithSSBFinalize)

	return m
}
