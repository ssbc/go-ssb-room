// SPDX-License-Identifier: MIT

// Package sqlite implements the SQLite backend of the admindb interfaces.
//
// It uses sql-migrate (github.com/rubenv/sql-migrate) for it's schema definition and maintainace.
// For query construction/ORM it uses SQLBoiler (https://github.com/volatiletech/sqlboiler).
//
// The process of updating the schema and ORM can be summarized as follows:
//
// 	1. Make changes to the interfaces in package admindb
//	2. Add a new migration to the 'migrations' folder
//	3. Run 'go test -run Simple', which applies all the migrations
//	4. Run sqlboiler to generate package models
//	5. Implement the interface as needed by using the models package
//
// For convenience step 3 and 4 are combined in the generate_models bash script.
package sqlite

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	migrate "github.com/rubenv/sql-migrate"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
)

type Database struct {
	db *sql.DB

	AuthWithSSB  roomdb.AuthWithSSBService
	AuthFallback roomdb.AuthFallbackService

	AllowList roomdb.AllowListService
	Aliases   roomdb.AliasService

	PinnedNotices roomdb.PinnedNoticesService
	Notices       roomdb.NoticesService

	Invites roomdb.InviteService
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

	// enable constraint enforcment for relations
	fname += "?_foreign_keys=on"

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

	if err := deleteConsumedInvites(db); err != nil {
		return nil, err
	}

	go func() { // server might not restart as often
		threeDays := 5 * 24 * time.Hour
		ticker := time.NewTicker(threeDays)
		for range ticker.C {
			err := transact(db, func(tx *sql.Tx) error {
				return deleteConsumedInvites(tx)
			})
			if err != nil {
				// TODO: hook up logging
				log.Printf("admindb: failed to clean up old invites: %s", err.Error())
			}
		}
	}()

	al := &AllowList{db}
	admindb := &Database{
		db:            db,
		AuthWithSSB:   AuthWithSSB{db},
		AuthFallback:  AuthFallback{db},
		AllowList:     al,
		Aliases:       Aliases{db},
		PinnedNotices: PinnedNotices{db},
		Notices:       Notices{db},

		Invites: Invites{
			db:        db,
			allowList: al,
		},
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
