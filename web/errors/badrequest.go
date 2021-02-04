package errors

import (
	"fmt"
)

type NotFound struct {
	What string
}

func (nf NotFound) Error() string {
	return fmt.Sprintf("rooms/web: item not found: %s", nf.What)
}

type BadRequest struct {
	Where   string
	Details error
}

func (br BadRequest) Error() string {
	return fmt.Sprintf("rooms/web: bad request error: %s", br.Details)
}
