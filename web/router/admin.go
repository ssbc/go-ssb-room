// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AdminDashboard = "admin:dashboard"
	AdminMenu      = "admin:menu"

	AdminAllowListOverview      = "admin:allow-list:overview"
	AdminAllowListAdd           = "admin:allow-list:add"
	AdminAllowListRemoveConfirm = "admin:allow-list:remove:confirm"
	AdminAllowListRemove        = "admin:allow-list:remove"

	AdminNoticeEdit = "admin:notice:edit"
	AdminNoticeSave = "admin:notice:save"
)

// Admin constructs a mux.Router containing the routes for the admin dashboard and settings pages
func Admin(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	m.Path("/dashboard").Methods("GET").Name(AdminDashboard)
	m.Path("/menu").Methods("GET").Name(AdminMenu)

	m.Path("/members").Methods("GET").Name(AdminAllowListOverview)
	m.Path("/members/add").Methods("POST").Name(AdminAllowListAdd)
	m.Path("/members/remove/confirm").Methods("GET").Name(AdminAllowListRemoveConfirm)
	m.Path("/members/remove").Methods("POST").Name(AdminAllowListRemove)

	m.Path("/notice/edit").Methods("GET").Name(AdminNoticeEdit)
	m.Path("/notice/save").Methods("POST").Name(AdminNoticeSave)

	return m
}
