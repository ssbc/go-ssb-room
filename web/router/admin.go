package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AdminDashboard = "admin:dashboard"
)

// Admin constructs a mux.Router containing the routes for the admin dashboard and settings pages
func Admin(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	// we dont strip path here because it somehow fucks with the middleware setup
	m.Path("/admin").Methods("GET").Name(AdminDashboard)
	// m.Path("/admin/settings").Methods("GET").Name(AdminSettings)

	return m
}
