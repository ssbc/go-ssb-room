package sqlite

import (
	"database/sql"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb"
)

// make sure to implement interfaces correctly
var _ admindb.AuthWithSSBService = (*AuthWithSSB)(nil)

type AuthWithSSB struct {
	db *sql.DB
}
