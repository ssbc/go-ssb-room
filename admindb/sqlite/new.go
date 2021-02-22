// SPDX-License-Identifier: MIT

package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
)

type Database struct {
	db *sql.DB

	AuthWithSSB  admindb.AuthWithSSBService
	AuthFallback admindb.AuthFallbackService

	AllowList admindb.AllowListService
	Aliases   admindb.AliasService

	PinnedNotices admindb.PinnedNoticesService
	Notices       admindb.NoticesService
}

// Open looks for a database file 'fname'
func Open(r repo.Interface) (*Database, error) {
	fname := r.GetPath("roomdb")

	if dir := filepath.Dir(fname); dir != "" {
		err := os.MkdirAll(dir, 0700)
		if err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("admindb: failed to create folder for database (%q): %w", dir, err)
		}
	}

	db, err := sql.Open("sqlite3", fname)
	if err != nil {
		return nil, fmt.Errorf("admindb: failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("admindb: sqlite ping failed: %w", err)
	}

	n, err := migrate.Exec(db, "sqlite3", migrationSource, migrate.Up)
	if err != nil {
		return nil, fmt.Errorf("admindb: failed to apply database mirations: %w", err)
	}
	if n > 0 {
		// TODO: hook up logging
		log.Printf("admindb: applied %d migrations", n)
	}

	admindb := &Database{
		db:            db,
		AuthWithSSB:   AuthWithSSB{db},
		AuthFallback:  AuthFallback{db},
		AllowList:     AllowList{db},
		Aliases:       Aliases{db},
		PinnedNotices: PinnedNotices{db},
		Notices:       Notices{db},
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
