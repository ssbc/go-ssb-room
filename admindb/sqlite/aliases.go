// SPDX-License-Identifier: MIT

package sqlite

import (
	"database/sql"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
)

// compiler assertion to ensure the struct fullfills the interface
var _ admindb.AliasService = (*Aliases)(nil)

type Aliases struct {
	db *sql.DB
}
