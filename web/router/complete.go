// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package router

import (
	"github.com/gorilla/mux"
)

// constant names for the named routes
const (
	CompleteIndex = "complete:index"

	CompleteNoticeShow = "complete:notice:show"
	CompleteNoticeList = "complete:notice:list"

	CompleteSetLanguage = "complete:set-language"

	CompleteAliasResolve = "complete:alias:resolve"

	CompleteInviteFacade         = "complete:invite:accept"
	CompleteInviteFacadeFallback = "complete:invite:accept:fallback"
	CompleteInviteInsertID       = "complete:invite:insert-id"
	CompleteInviteConsume        = "complete:invite:consume"

	MembersChangePasswordForm = "members:change-password:form"
	MembersChangePassword     = "members:change-password"

	OpenModeCreateInvite = "open:invites:create"
)

// CompleteApp constructs a mux.Router containing the routes for batch Complete html frontend
func CompleteApp() *mux.Router {
	m := mux.NewRouter()

	Auth(m)
	Admin(m.PathPrefix("/admin").Subrouter())

	m.Path("/").Methods("GET").Name(CompleteIndex)

	m.Path("/alias/{alias}").Methods("GET").Name(CompleteAliasResolve)

	m.Path("/members/change-password").Methods("GET").Name(MembersChangePasswordForm)
	m.Path("/members/change-password").Methods("POST").Name(MembersChangePassword)

	m.Path("/create-invite").Methods("GET", "POST").Name(OpenModeCreateInvite)
	m.Path("/join").Methods("GET").Name(CompleteInviteFacade)
	m.Path("/join-fallback").Methods("GET").Name(CompleteInviteFacadeFallback)
	m.Path("/join-manually").Methods("GET").Name(CompleteInviteInsertID)
	m.Path("/invite/consume").Methods("POST").Name(CompleteInviteConsume)

	m.Path("/notice/show").Methods("GET").Name(CompleteNoticeShow)
	m.Path("/notice/list").Methods("GET").Name(CompleteNoticeList)

	m.Path("/set-language").Methods("POST").Name(CompleteSetLanguage)

	return m
}
