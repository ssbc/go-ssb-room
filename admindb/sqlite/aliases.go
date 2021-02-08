package sqlite

import (
	"database/sql"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb"
)

// make sure to implement interfaces correctly
var _ admindb.AliasService = (*Aliases)(nil)

type Aliases struct {
	db *sql.DB
}
