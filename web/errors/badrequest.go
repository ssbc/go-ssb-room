// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

// Package errors defines some well defined errors, like incomplete/wrong request data or object not found(404), for the purpose of internationalization.
package errors

import (
	"errors"
	"fmt"
)

var (
	ErrNotAuthorized = errors.New("rooms/web: not authorized")

	ErrDenied = errors.New("rooms: this key has been banned")
)

// ErrGenericLocalized is used for one-off errors that primarily are presented for the user.
// The contained label is passed to the i18n engine for translation.
type ErrGenericLocalized struct{ Label string }

func (err ErrGenericLocalized) Error() string {
	return fmt.Sprintf("rooms/web: localized error (%s)", err.Label)
}

type ErrNotFound struct{ What string }

func (nf ErrNotFound) Error() string {
	return fmt.Sprintf("rooms/web: item not found: %s", nf.What)
}

type ErrBadRequest struct {
	Where   string
	Details error
}

func (err ErrBadRequest) Unwrap() error {
	return err.Details
}

func (br ErrBadRequest) Error() string {
	return fmt.Sprintf("rooms/web: bad request error: %s", br.Details)
}

type ErrForbidden struct{ Details error }

func (f ErrForbidden) Error() string {
	return fmt.Sprintf("rooms/web: access denied: %s", f.Details)
}

// ErrRedirect is used when the controller decides to not render a page
type ErrRedirect struct {
	Path string

	// reason will be added as a flash error
	Reason error
}

func (err ErrRedirect) Unwrap() error {
	return err.Reason
}

func (err ErrRedirect) Error() string {
	return fmt.Sprintf("rooms/web: redirecting to: %s (reason: %s)", err.Path, err.Reason)
}

type PageNotFound struct{ Path string }

func (e PageNotFound) Error() string {
	return fmt.Sprintf("rooms/web: page not found: %s", e.Path)
}

type DatabaseError struct{ Reason error }

func (e DatabaseError) Error() string {
	return fmt.Sprintf("rooms/web: database failed to complete query: %s", e.Reason.Error())
}
