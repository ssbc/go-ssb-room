// SPDX-License-Identifier: MIT

// Package errors defines some well defined errors, like incomplete/wrong request data or object not found(404), for the purpose of internationalization.
package errors

import (
	"fmt"
)

type ErrNotFound struct {
	What string
}

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
