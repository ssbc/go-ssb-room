// SPDX-License-Identifier: MIT

package router

import (
	"github.com/gorilla/mux"
)

// constant names for the named routes
const (
	CompleteIndex  = "complete:index"
	CompleteAbout  = "complete:about"
	CompleteNotice = "complete:notice"
)

// CompleteApp constructs a mux.Router containing the routes for batch Complete html frontend
func CompleteApp() *mux.Router {
	m := mux.NewRouter()

	Auth(m.PathPrefix("/auth").Subrouter())
	Admin(m.PathPrefix("/admin").Subrouter())
	News(m.PathPrefix("/news").Subrouter())

	m.Path("/").Methods("GET").Name(CompleteIndex)
	m.Path("/about").Methods("GET").Name(CompleteAbout)
	m.Path("/notice").Methods("GET").Name(CompleteNotice)

	return m
}
