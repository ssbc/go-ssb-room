package errors

import (
	"fmt"
)

type BadRequest struct {
	Where   string
	Details error
}

func (br BadRequest) Error() string {
	return fmt.Sprintf("rooms: bad request error: %s", br.Details)
}
