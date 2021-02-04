package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AuthSignIn  = "Auth:SignIn"
	AuthSignOut = "Auth:SignOut"
)

// NewSignin constructs a mux.Router containing the routes for sign-in and -out
func Auth(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	m.Path("/signIn").Methods("GET").Name(AuthSignIn)
	m.Path("/signOut").Methods("GET").Name(AuthSignOut)

	return m
}
