// SPDX-License-Identifier: MIT

package sqlite

import (
	"database/sql"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
)

// compiler assertion to ensure the struct fullfills the interface
var _ admindb.AuthWithSSBService = (*AuthWithSSB)(nil)

type AuthWithSSB struct {
	db *sql.DB
}
