// SPDX-License-Identifier: MIT

package sqlite

import (
	"database/sql"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.AuthWithSSBService = (*AuthWithSSB)(nil)

type AuthWithSSB struct {
	db *sql.DB
}
