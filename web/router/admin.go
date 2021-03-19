// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	AdminDashboard = "admin:dashboard"
	AdminMenu      = "admin:menu"

	AdminAliasesOverview      = "admin:aliases:overview"
	AdminAliasesRevokeConfirm = "admin:aliases:revoke:confirm"
	AdminAliasesRevoke        = "admin:aliases:revoke"

	AdminDeniedKeysOverview      = "admin:denied-keys:overview"
	AdminDeniedKeysAdd           = "admin:denied-keys:add"
	AdminDeniedKeysRemoveConfirm = "admin:denied-keys:remove:confirm"
	AdminDeniedKeysRemove        = "admin:denied-keys:remove"

	AdminMembersOverview      = "admin:members:overview"
	AdminMembersAdd           = "admin:members:add"
	AdminMembersRemoveConfirm = "admin:members:remove:confirm"
	AdminMembersRemove        = "admin:members:remove"

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

	m.Path("/aliases").Methods("GET").Name(AdminAliasesOverview)
	m.Path("/aliases/revoke/confirm").Methods("GET").Name(AdminAliasesRevokeConfirm)
	m.Path("/aliases/revoke").Methods("POST").Name(AdminAliasesRevoke)

	m.Path("/denied").Methods("GET").Name(AdminDeniedKeysOverview)
	m.Path("/denied/add").Methods("POST").Name(AdminDeniedKeysAdd)
	m.Path("/denied/remove/confirm").Methods("GET").Name(AdminDeniedKeysRemoveConfirm)
	m.Path("/denied/remove").Methods("POST").Name(AdminDeniedKeysRemove)

	m.Path("/members").Methods("GET").Name(AdminMembersOverview)
	m.Path("/members/add").Methods("POST").Name(AdminMembersAdd)
	m.Path("/members/remove/confirm").Methods("GET").Name(AdminMembersRemoveConfirm)
	m.Path("/members/remove").Methods("POST").Name(AdminMembersRemove)

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
