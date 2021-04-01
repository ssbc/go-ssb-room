// SPDX-License-Identifier: MIT

// Package errors defines some well defined errors, like incomplete/wrong request data or object not found(404), for the purpose of internationalization.
package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotAuthorized = errors.New("rooms/web: not authorized")
	ErrDenied        = errors.New("rooms: this key has been banned")
)

type ErrNotFound struct{ What string }

func (nf ErrNotFound) Error() string {
	return fmt.Sprintf("rooms/web: item not found: %s", nf.What)
}

type ErrBadRequest struct {
	Where   string
	Details error
}

func (br ErrBadRequest) Error() string {
	return fmt.Sprintf("rooms/web: bad request error: %s", br.Details)
}

type ErrForbidden struct{ Details error }

func (f ErrForbidden) Error() string {
	return fmt.Sprintf("rooms/web: access denied: %s", f.Details)
}

// ErrRedirect decide to not render a page during the controller
type ErrRedirect struct{ Path string }

func (err ErrRedirect) Error() string {
	return fmt.Sprintf("rooms/web: redirecting to: %s", err.Path)
}

type PageNotFound struct{ Path string }

func (e PageNotFound) Error() string {
	return fmt.Sprintf("rooms/web: page not found: %s", e.Path)
}

type DatabaseError struct{ Reason error }

func (e DatabaseError) Error() string {
	return fmt.Sprintf("rooms/web: database failed to complete query: %s", e.Reason.Error())
}
