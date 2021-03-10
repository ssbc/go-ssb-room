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

	AdminInvitesOverview      = "admin:invites:overview"
	AdminInvitesRevokeConfirm = "admin:invites:revoke:confirm"
	AdminInvitesRevoke        = "admin:invites:revoke"
	AdminInvitesCreate        = "admin:invites:create"

	AdminNoticeEdit             = "admin:notice:edit"
	AdminNoticeSave             = "admin:notice:save"
	AdminNoticeDraftTranslation = "admin:notice:translation:draft"
	AdminNoticeAddTranslation   = "admin:notice:translation:add"
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
	m.Path("/notice/translation/draft").Methods("GET").Name(AdminNoticeDraftTranslation)
	m.Path("/notice/translation/add").Methods("POST").Name(AdminNoticeAddTranslation)
	m.Path("/notice/save").Methods("POST").Name(AdminNoticeSave)

	m.Path("/invites").Methods("GET").Name(AdminInvitesOverview)
	m.Path("/invites/revoke/confirm").Methods("GET").Name(AdminInvitesRevokeConfirm)
	m.Path("/invites/revoke").Methods("POST").Name(AdminInvitesRevoke)
	m.Path("/invites/create").Methods("POST").Name(AdminInvitesCreate)

	return m
}
