// SPDX-License-Identifier: MIT

package router

import "github.com/gorilla/mux"

// constant names for the named routes
const (
	NewsOverview = "news:overview"
	NewsPost     = "news:post"
)

// News constructs a mux.Router containing the routes for news aspects of the site
func News(m *mux.Router) *mux.Router {
	if m == nil {
		m = mux.NewRouter()
	}

	m.Path("/").Methods("GET").Name(NewsOverview)
	m.Path("/post").Methods("GET").Name(NewsPost)

	return m
}
