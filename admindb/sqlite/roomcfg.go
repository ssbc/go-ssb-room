// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/friendsofgo/errors"
	"github.com/mattn/go-sqlite3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb/sqlite/models"
	refs "go.mindeco.de/ssb-refs"
)

// compiler assertion to ensure the struct fullfills the interface
var _ admindb.AllowListService = (*AllowList)(nil)

type AllowList struct {
	db *sql.DB
}

// Add adds the feed to the list.
func (l AllowList) Add(ctx context.Context, a refs.FeedRef) error {
	// single insert transaction but this makes it easier to re-use in invites.Consume
	return transact(l.db, func(tx *sql.Tx) error {
		return l.add(ctx, tx, a)
	})
}

// this add is not exported and for internal use with transactions.
func (l AllowList) add(ctx context.Context, tx *sql.Tx, a refs.FeedRef) error {
	// TODO: better valid
	if _, err := refs.ParseFeedRef(a.Ref()); err != nil {
		return err
	}

	var entry models.AllowList

	entry.PubKey.FeedRef = a
	err := entry.Insert(ctx, tx, boil.Whitelist("pub_key"))
	if err != nil {
		var sqlErr sqlite3.Error
		if errors.As(err, &sqlErr) && sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique {
			return admindb.ErrAlreadyAdded{Ref: a}
		}

		return fmt.Errorf("allow-list: failed to insert new entry %s: %w - type:%T", entry.PubKey, err, err)
	}

	return nil
}

// HasFeed returns true if a feed is on the list.
func (l AllowList) HasFeed(ctx context.Context, h refs.FeedRef) bool {
	_, err := models.AllowLists(qm.Where("pub_key = ?", h.Ref())).One(ctx, l.db)
	if err != nil {
		return false
	}
	return true
}

// HasID returns true if a feed is on the list.
func (l AllowList) HasID(ctx context.Context, id int64) bool {
	_, err := models.FindAllowList(ctx, l.db, id)
	if err != nil {
		return false
	}
	return true
}

// GetByID returns the entry if a feed with that ID is on the list.
func (l AllowList) GetByID(ctx context.Context, id int64) (admindb.ListEntry, error) {
	var le admindb.ListEntry
	entry, err := models.FindAllowList(ctx, l.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return le, admindb.ErrNotFound
		}
		return le, err
	}

	le.ID = entry.ID
	le.PubKey = entry.PubKey.FeedRef

	return le, nil
}

// List returns a list of all the feeds.
func (l AllowList) List(ctx context.Context) (admindb.ListEntries, error) {
	all, err := models.AllowLists().All(ctx, l.db)
	if err != nil {
		return nil, err
	}

	var asRefs = make(admindb.ListEntries, len(all))
	for i, allowed := range all {

		asRefs[i] = admindb.ListEntry{
			ID:     allowed.ID,
			PubKey: allowed.PubKey.FeedRef,
		}
	}

	return asRefs, nil
}

// RemoveFeed removes the feed from the list.
func (l AllowList) RemoveFeed(ctx context.Context, r refs.FeedRef) error {
	entry, err := models.AllowLists(qm.Where("pub_key = ?", r.Ref())).One(ctx, l.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admindb.ErrNotFound
		}
		return err
	}

	_, err = entry.Delete(ctx, l.db)
	if err != nil {
		return err
	}

	return nil
}

// RemoveID removes the feed from the list.
func (l AllowList) RemoveID(ctx context.Context, id int64) error {
	entry, err := models.FindAllowList(ctx, l.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admindb.ErrNotFound
		}
		return err
	}

	_, err = entry.Delete(ctx, l.db)
	if err != nil {
		return err
	}

	return nil
}
