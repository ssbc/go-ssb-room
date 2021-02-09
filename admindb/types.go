// SPDX-License-Identifier: MIT

package admindb

// It's important to wrap all the model generated types into these since we don't want the admindb interfaces to depend on them.

// User holds all the information an authenticated user of the site has.
type User struct {
	ID   int64
	Name string
}
