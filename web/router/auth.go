package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AuthFallbackSignInForm = "Auth:Fallback:Form:SignIn"
	AuthFallbackSignIn     = "Auth:Fallback:SignIn"
	AuthFallbackSignOut    = "Auth:Fallback:SignOut"

	AuthWithSSBSignIn  = "Auth:WithSSB:SignIn"
	AuthWithSSBSignOut = "Auth:WithSSB:SignOut"
)

// NewSignin constructs a mux.Router containing the routes for sign-in and -out
func Auth(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	// register fallback
	m.Path("/fallback/signin").Methods("GET").Name(AuthFallbackSignInForm)
	m.Path("/fallback/signin").Methods("POST").Name(AuthFallbackSignIn)
	m.Path("/fallback/signOut").Methods("GET").Name(AuthFallbackSignOut)

	m.Path("/withssb/signIn").Methods("GET").Name(AuthWithSSBSignIn)
	m.Path("/withssb/signOut").Methods("GET").Name(AuthWithSSBSignOut)

	return m
}
