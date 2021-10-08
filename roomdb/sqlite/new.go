// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

// Package sqlite implements the SQLite backend of the roomdb interfaces.
//
// It uses sql-migrate (github.com/rubenv/sql-migrate) for it's schema definition and maintenance.
// For query construction/ORM it uses SQLBoiler (https://github.com/volatiletech/sqlboiler).
//
// The process of updating the schema and ORM can be summarized as follows:
//
// 	1. Make changes to the interfaces in package roomdb
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

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
)

type Database struct {
	db *sql.DB

	AuthFallback AuthFallback
	AuthWithSSB  AuthWithSSB

	Members Members
	Aliases Aliases
	Invites Invites
	Config  Config

	DeniedKeys DeniedKeys

	PinnedNotices PinnedNotices
	Notices       Notices
}

// Open looks for a database file 'fname'
func Open(r repo.Interface) (*Database, error) {
	fname := r.GetPath("roomdb")

	if dir := filepath.Dir(fname); dir != "" {
		err := os.MkdirAll(dir, 0700)
		if err != nil && !os.IsExist(err) {
			return nil, fmt.Errorf("roomdb: failed to create folder for database (%q): %w", dir, err)
		}
	}

	// enable constraint enforcment for relations
	fname += "?_foreign_keys=on"

	db, err := sql.Open("sqlite3", fname)
	if err != nil {
		return nil, fmt.Errorf("roomdb: failed to open sqlite database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("roomdb: sqlite ping failed: %w", err)
	}

	n, err := migrate.Exec(db, "sqlite3", migrationSource, migrate.Up)
	if err != nil {
		return nil, fmt.Errorf("roomdb: failed to apply database mirations: %w", err)
	}
	if n > 0 {
		// TODO: hook up logging
		log.Printf("roomdb: applied %d migrations", n)
	}

	if err := deleteConsumedInvites(db); err != nil {
		return nil, err
	}

	if err := deleteConsumedResetTokens(db); err != nil {
		return nil, err
	}

	// scrub old invites and reset tokens
	go func() { // server might not restart as often
		fiveDays := 5 * 24 * time.Hour
		ticker := time.NewTicker(fiveDays)
		for range ticker.C {
			err := transact(db, func(tx *sql.Tx) error {
				if err := deleteConsumedResetTokens(tx); err != nil {
					return err
				}
				return deleteConsumedInvites(tx)
			})
			if err != nil {
				// TODO: hook up logging
				log.Printf("roomdb: failed to clean up old invites: %s", err.Error())
			}
		}
	}()

	ml := Members{db}

	roomdb := &Database{
		db: db,

		Aliases:       Aliases{db},
		AuthFallback:  AuthFallback{db},
		AuthWithSSB:   AuthWithSSB{db},
		Config:        Config{db},
		DeniedKeys:    DeniedKeys{db},
		Invites:       Invites{db: db, members: ml},
		Notices:       Notices{db},
		Members:       ml,
		PinnedNotices: PinnedNotices{db},
	}

	return roomdb, nil
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
