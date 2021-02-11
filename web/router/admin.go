// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AdminDashboard = "admin:dashboard"

	AdminAllowListOverview      = "admin:allow-list:overview"
	AdminAllowListRemoveAdd     = "admin:allow-list:add"
	AdminAllowListRemoveAsk     = "admin:allow-list:remove:ask"
	AdminAllowListRemoveConfirm = "admin:allow-list:remove:confirmed"
)

// Admin constructs a mux.Router containing the routes for the admin dashboard and settings pages
func Admin(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	m.Path("/dashboard").Methods("GET").Name(AdminDashboard)

	m.Path("/allow-list").Methods("GET").Name(AdminAllowListOverview)
	m.Path("/allow-list/add").Methods("POST").Name(AdminAllowListRemoveAdd)
	m.Path("/allow-list/remove").Methods("GET").Name(AdminAllowListRemoveAsk)
	m.Path("/allow-list/remove").Methods("POST").Name(AdminAllowListRemoveConfirm)

	// m.Path("/settings").Methods("GET").Name(AdminSettings)

	return m
}
