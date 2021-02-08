package sqlite

import (
	"database/sql"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb"
)

// make sure to implement interfaces correctly
var _ admindb.RoomService = (*Rooms)(nil)

type Rooms struct {
	db *sql.DB
}
