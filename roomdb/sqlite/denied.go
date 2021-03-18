// SPDX-License-Identifier: MIT

// +build ignore

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/friendsofgo/errors"
	"github.com/mattn/go-sqlite3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
	refs "go.mindeco.de/ssb-refs"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.DeniedListService = (*DeniedList)(nil)

// The DeniedList is backed by the members table
type DeniedList struct {
	db *sql.DB
}

// Add adds the feed to the list.
func (al DeniedList) Add(ctx context.Context, a refs.FeedRef) error {
	// single insert transaction but this makes it easier to re-use in invites.Consume
	return transact(al.db, func(tx *sql.Tx) error {
		return al.add(ctx, tx, a)
	})
}

// this add is not exported and for internal use with transactions.
func (al DeniedList) add(ctx context.Context, tx *sql.Tx, a refs.FeedRef) error {
	// TODO: better valid
	if _, err := refs.ParseFeedRef(a.Ref()); err != nil {
		return err
	}

	var entry models.Member
	entry.PubKey.FeedRef = a

	err := entry.Insert(ctx, tx, boil.Whitelist("pub_key"))
	if err != nil {
		var sqlErr sqlite3.Error
		if errors.As(err, &sqlErr) && sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return roomdb.ErrAlreadyAdded{Ref: a}
		}

		return fmt.Errorf("Denied-list: failed to insert new entry %s: %w - type:%T", entry.PubKey, err, err)
	}

	return nil
}

// HasFeed returns true if a feed is on the list.
func (al DeniedList) HasFeed(ctx context.Context, h refs.FeedRef) bool {
	_, err := models.DeniedLists(qm.Where("pub_key = ?", h.Ref())).One(ctx, al.db)
	if err != nil {
		return false
	}
	return true
}

// HasID returns true if a feed is on the list.
func (al DeniedList) HasID(ctx context.Context, id int64) bool {
	_, err := models.FindDeniedList(ctx, al.db, id)
	if err != nil {
		return false
	}
	return true
}

// GetByID returns the entry if a feed with that ID is on the list.
func (al DeniedList) GetByID(ctx context.Context, id int64) (roomdb.ListEntry, error) {
	var le roomdb.ListEntry
	entry, err := models.FindDeniedList(ctx, al.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return le, roomdb.ErrNotFound
		}
		return le, err
	}

	le.ID = entry.ID
	le.PubKey = entry.PubKey.FeedRef

	return le, nil
}

// List returns a list of all the feeds.
func (al DeniedList) List(ctx context.Context) (roomdb.ListEntries, error) {
	all, err := models.DeniedLists().All(ctx, al.db)
	if err != nil {
		return nil, err
	}

	var asRefs = make(roomdb.ListEntries, len(all))
	for i, Denieded := range all {
		asRefs[i] = roomdb.ListEntry{
			ID:     Denieded.ID,
			PubKey: Denieded.PubKey.FeedRef,
		}
	}

	return asRefs, nil
}

// RemoveFeed removes the feed from the list.
func (al DeniedList) RemoveFeed(ctx context.Context, r refs.FeedRef) error {
	entry, err := models.DeniedLists(qm.Where("pub_key = ?", r.Ref())).One(ctx, al.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return roomdb.ErrNotFound
		}
		return err
	}

	_, err = entry.Delete(ctx, al.db)
	if err != nil {
		return err
	}

	return nil
}

// RemoveID removes the feed from the list.
func (al DeniedList) RemoveID(ctx context.Context, id int64) error {
	entry, err := models.FindDeniedList(ctx, al.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return roomdb.ErrNotFound
		}
		return err
	}

	_, err = entry.Delete(ctx, al.db)
	if err != nil {
		return err
	}

	return nil
}
