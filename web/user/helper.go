// Package user implements helpers for accessing the currently logged in admin or moderator of an active request.
package user

import (
	"context"
	"net/http"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"go.mindeco.de/http/auth"
)

type roomUserContextKeyType string

var roomUserContextKey roomUserContextKeyType = "ssb:room:httpcontext:user"

 // FromContext returns the user or nil if not logged in
func FromContext(ctx context.Context) *roomdb.User {
	v := ctx.Value(roomUserContextKey)

	user, ok := v.(*roomdb.User)
	if !ok {
		return nil
	}

	return user
}

// ContextInjecter returns middleware for injecting a user id into the request context
func ContextInjecter(fs roomdb.AuthFallbackService, a *auth.Handler) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			v, err := a.AuthenticateRequest(req)
			if err != nil {
				next.ServeHTTP(w, req)
				return
			}

			uid, ok := v.(int64)
			if !ok {
				next.ServeHTTP(w, req)
				return
			}

			user, err := fs.GetByID(req.Context(), uid)
			if err != nil {
				next.ServeHTTP(w, req)
				return
			}

			ctx := context.WithValue(req.Context(), roomUserContextKey, user)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

// TemplateHelper returns a function to be used with the http/render package.
// It has to return a function twice because the first is evaluated with the request before it gets passed onto html/template's FuncMap.
func TemplateHelper() func(*http.Request) interface{} {
	return func(r *http.Request) interface{} {
		no := func() *roomdb.User { return nil }

		user := FromContext(r.Context())
		if user == nil {
			return no
		}

		yes := func() *roomdb.User { return user }
		return yes
	}
}
