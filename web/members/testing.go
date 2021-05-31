// SPDX-License-Identifier: MIT

package members

import (
	"context"
	"net/http"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
)

// MiddlewareForTests gives us a way to inject _test members_. It should not be used in production.
// This is part of testing.go because we need to use roomMemberContextKey, which shouldn't be exported either.
// TODO: could be protected with an extra build tag.
// (Sadly +build test does not exist https://github.com/golang/go/issues/21360 )
func MiddlewareForTests(m *roomdb.Member) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), roomMemberContextKey, m)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}
