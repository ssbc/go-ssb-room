// SPDX-License-Identifier: MIT

// Package members implements helpers for accessing the currently logged in admin or moderator of an active request.
package members

import (
	"context"
	"net/http"

	"go.mindeco.de/http/auth"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	authWithSSB "github.com/ssb-ngi-pointer/go-ssb-room/web/handlers/auth"
)

type roomMemberContextKeyType string

var roomMemberContextKey roomMemberContextKeyType = "ssb:room:httpcontext:member"

type Middleware func(next http.Handler) http.Handler

// AuthenticateFromContext calls the next http handler if there is a member stored in the context
// otherwise it will call r.Error
func AuthenticateFromContext(r *render.Renderer) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			if FromContext(req.Context()) == nil {
				r.Error(w, req, http.StatusUnauthorized, weberrors.ErrNotAuthorized)
				return
			}
			next.ServeHTTP(w, req)
		})
	}
}

// FromContext returns the member or nil if not logged in
func FromContext(ctx context.Context) *roomdb.Member {
	v := ctx.Value(roomMemberContextKey)

	m, ok := v.(*roomdb.Member)
	if !ok {
		return nil
	}

	return m
}

// ContextInjecter returns middleware for injecting a member into the context of the request.
// Retreive it using FromContext(ctx)
func ContextInjecter(mdb roomdb.MembersService, withPassword *auth.Handler, withSSB *authWithSSB.WithSSBHandler) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
			var (
				member *roomdb.Member

				errWithPassword, errWithSSB error
			)

			v, errWithPassword := withPassword.AuthenticateRequest(req)
			if errWithPassword == nil {
				mid, ok := v.(int64)
				if !ok {
					next.ServeHTTP(w, req)
					return
				}

				m, err := mdb.GetByID(req.Context(), mid)
				if err != nil {
					next.ServeHTTP(w, req)
					return
				}
				member = &m
			}

			m, errWithSSB := withSSB.AuthenticateRequest(req)
			if errWithSSB == nil {
				member = m
			}

			// if both methods failed, don't update the context
			if errWithPassword != nil && errWithSSB != nil {
				next.ServeHTTP(w, req)
				return
			}

			ctx := context.WithValue(req.Context(), roomMemberContextKey, member)
			next.ServeHTTP(w, req.WithContext(ctx))
		})
	}
}

// TemplateHelpers returns functions to be used with the go.mindeco.de/http/render package.
// Each has to return a function twice because the first is evaluated with the request before it gets passed onto html/template's FuncMap.
//
//  {{ is_logged_in }} returns true or false depending if the user is logged in
//
//  {{ member_has_role "string" }} returns a boolean which confrms wether the member has a certain role
//
//  {{ member_is_admin }} is a shortcut for {{ member_has_role "admin" }}
func TemplateHelpers() []render.Option {

	return []render.Option{
		render.InjectTemplateFunc("is_logged_in", func(r *http.Request) interface{} {
			no := func() *roomdb.Member { return nil }

			member := FromContext(r.Context())
			if member == nil {
				return no
			}

			yes := func() *roomdb.Member { return member }
			return yes
		}),

		render.InjectTemplateFunc("member_has_role", func(r *http.Request) interface{} {
			no := func(_ string) bool { return false }

			member := FromContext(r.Context())
			if member == nil {
				return no
			}

			return func(has string) bool {
				var r roomdb.Role
				if err := r.UnmarshalText([]byte(has)); err != nil {
					return false
				}
				return member.Role == r
			}
		}),

		render.InjectTemplateFunc("member_is_admin", func(r *http.Request) interface{} {
			no := func() bool { return false }

			member := FromContext(r.Context())
			if member == nil {
				return no
			}

			return func() bool {
				return member.Role == roomdb.RoleAdmin
			}
		}),
	}
}
