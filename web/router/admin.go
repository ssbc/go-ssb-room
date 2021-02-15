// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AdminDashboard = "admin:dashboard"

	AdminAllowListOverview      = "admin:allow-list:overview"
	AdminAllowListAdd           = "admin:allow-list:add"
	AdminAllowListRemoveConfirm = "admin:allow-list:remove:confirm"
	AdminAllowListRemove        = "admin:allow-list:remove"
)

// Admin constructs a mux.Router containing the routes for the admin dashboard and settings pages
func Admin(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	m.Path("/dashboard").Methods("GET").Name(AdminDashboard)

	m.Path("/allow-list").Methods("GET").Name(AdminAllowListOverview)
	m.Path("/allow-list/add").Methods("POST").Name(AdminAllowListAdd)
	m.Path("/allow-list/remove/confirm").Methods("GET").Name(AdminAllowListRemoveConfirm)
	m.Path("/allow-list/remove").Methods("POST").Name(AdminAllowListRemove)

	return m
}
