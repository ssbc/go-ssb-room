package user

import (
	"context"
	"net/http"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
)

// MiddlewareForTests gives us a way to inject _test users_. It should not be used in production.
// This is exists here because we need to use roomUserContextKey which shouldn't be exported either.
// TODO: could be protected with an extra build tag.
// (Sadly +build test does not exist https://github.com/golang/go/issues/21360 )
func MiddlewareForTests(user *admindb.User) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			ctx := context.WithValue(req.Context(), roomUserContextKey, user)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}
