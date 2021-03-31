// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
)

// cblgh: ask cryptix about the details of the syntax of the "compiler assertion" below
// why two parens? this does not look like a typical type assertion? e.g. <var>.(type)
// hm-maybe this is a type conversion, forcing "nil" to be a *Aliases?
var _ roomdb.AliasesService = (*Aliases)(nil)

// the database will only ever store one row, which contains all the room settings
const configRowID = 0

/* Config basically enables long-term memory for the server when it comes to storing settings. Currently, the only
* stored settings is the privacy mode of the room.
*/
type Config struct {
	db *sql.DB
}

// cblgh questions:
// * is storing the entire config in a single row really ugly? ._.
func (c Config) GetPrivacyMode(ctx context.Context) (roomdb.PrivacyMode, error) {
	config, err := models.FindConfig(ctx, c.db, configRowID)
	if err != nil {
		return roomdb.ModeUnknown, err
	}

	// use a type conversion to tell compiler the returned value is a roomdb.PrivacyMode
	pm := (roomdb.PrivacyMode)(config.PrivacyMode)
	err = pm.IsValid()
	if err != nil {
		return roomdb.ModeUnknown, err
	}

	return pm, nil
}
