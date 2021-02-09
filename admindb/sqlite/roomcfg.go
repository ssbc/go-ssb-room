// SPDX-License-Identifier: MIT

package sqlite

import (
	"database/sql"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
)

// make sure to implement interfaces correctly
var _ admindb.RoomService = (*Rooms)(nil)

type Rooms struct {
	db *sql.DB
}
