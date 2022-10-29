// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"

	"github.com/friendsofgo/errors"
	"github.com/mattn/go-sqlite3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/roomdb/sqlite/models"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.AliasesService = (*Aliases)(nil)

type Aliases struct {
	db *sql.DB
}

// Resolve returns all the relevant information for that alias or an error if it doesnt exist
func (a Aliases) Resolve(ctx context.Context, name string) (roomdb.Alias, error) {
	return a.findOne(ctx, qm.Where("name = ?", name))
}

// GetByID returns the alias for that ID or an error
func (a Aliases) GetByID(ctx context.Context, id int64) (roomdb.Alias, error) {
	return a.findOne(ctx, qm.Where("id = ?", id))
}

func (a Aliases) findOne(ctx context.Context, by qm.QueryMod) (roomdb.Alias, error) {
	var found roomdb.Alias

	// construct query which resolves the Member relation and by which we shoudl look for it
	qry := append([]qm.QueryMod{qm.Load("Member")}, by)

	entry, err := models.Aliases(qry...).One(ctx, a.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return found, roomdb.ErrNotFound
		}
		return found, err
	}

	// unpack models into roomdb type
	found.ID = entry.ID
	found.Name = entry.Name
	found.Signature = entry.Signature
	found.Feed = entry.R.Member.PubKey.FeedRef

	return found, nil
}

// List returns a list of all registerd aliases
func (a Aliases) List(ctx context.Context) ([]roomdb.Alias, error) {
	all, err := models.Aliases(qm.Load("Member")).All(ctx, a.db)
	if err != nil {
		return nil, err
	}

	var aliases = make([]roomdb.Alias, len(all))
	for i, entry := range all {
		aliases[i] = roomdb.Alias{
			ID:        entry.ID,
			Name:      entry.Name,
			Feed:      entry.R.Member.PubKey.FeedRef,
			Signature: entry.Signature,
		}
	}

	return aliases, nil
}

// Register receives an alias and signature for it. Validation needs to happen before this.
func (a Aliases) Register(ctx context.Context, alias string, userFeed refs.FeedRef, signature []byte) error {
	return transact(a.db, func(tx *sql.Tx) error {
		// check we have a members entry for the feed and load it to get its ID
		memberEntry, err := models.Members(qm.Where("pub_key = ?", userFeed.Ref())).One(ctx, tx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return roomdb.ErrNotFound
			}
			return err
		}

		var newEntry models.Alias
		newEntry.Name = alias
		newEntry.MemberID = memberEntry.ID
		newEntry.Signature = signature

		err = newEntry.Insert(ctx, tx, boil.Infer())
		var sqlErr sqlite3.Error
		if errors.As(err, &sqlErr) && sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return roomdb.ErrAliasTaken{Name: alias}
		}
		return err
	})
}

// Revoke removes an alias from the system
func (a Aliases) Revoke(ctx context.Context, alias string) error {
	return transact(a.db, func(tx *sql.Tx) error {
		qry := append([]qm.QueryMod{qm.Load("Member")}, qm.Where("name = ?", alias))

		entry, err := models.Aliases(qry...).One(ctx, a.db)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return roomdb.ErrNotFound
			}
			return err
		}

		_, err = entry.Delete(ctx, tx)
		return err
	})
}
