// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
	"github.com/volatiletech/sqlboiler/v4/boil"
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

func (c Config) SetPrivacyMode(ctx context.Context, pm roomdb.PrivacyMode) error {
	fmt.Println("setting privacy mode!!")
	// make sure the privacy mode is an ok value
	err := pm.IsValid()
	if err != nil {
		return err
	}

	// cblgh: a walkthrough of this step (again, now that i have some actual context) would be real good :)
	err = transact(c.db, func(tx *sql.Tx) error {
		// get the settings row
		config, err := models.FindConfig(ctx, c.db, configRowID)
		if err != nil {
			return err
		}

		// set the new privacy mode
		config.PrivacyMode = int64(pm)
		// issue update stmt
		rowsAffected, err := config.Update(ctx, tx, boil.Infer())
		if err != nil {
			return err
		}
		if rowsAffected == 0 {
			return fmt.Errorf("setting privacy mode should have update the settings row, instead 0 rows were updated")
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil // alles gut!!
}
