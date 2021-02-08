package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb"
)

type Database struct {
	db *sql.DB

	AuthWithSSB  admindb.AuthWithSSBService
	AuthFallback admindb.AuthFallbackService

	Rooms   admindb.RoomService
	Aliases admindb.AliasService
}

// Open looks for a database file 'fname'
func Open(fname string) (*Database, error) {
	db, err := sql.Open("sqlite3", fname)
	if err != nil {
		return nil, fmt.Errorf("sqlite/open failed: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("sqlite/open: ping failed: %w", err)
	}

	admindb := &Database{
		db:           db,
		AuthWithSSB:  AuthWithSSB{db},
		AuthFallback: AuthFallback{db},
		Rooms:        Rooms{db},
		Aliases:      Aliases{db},
	}

	return admindb, nil
}

// Close closes the contained sql database object
func (t Database) Close() error {
	return t.db.Close()
}

func transact(db *sql.DB, fn func(tx *sql.Tx) error) error {
	var err error
	var tx *sql.Tx
	tx, err = db.Begin()
	if err != nil {
		return fmt.Errorf("transact: could not begin transaction: %w", err)
	}
	if err = fn(tx); err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			err = fmt.Errorf("rollback failed after %s: %s", err.Error(), err2.Error())
		} else {
			err = fmt.Errorf("transaction failed, rolling back: %w", err)
		}
		return err
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("transact: could not commit transaction: %w", err)
	}

	return nil
}
