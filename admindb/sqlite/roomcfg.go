// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb/sqlite/models"
	refs "go.mindeco.de/ssb-refs"
)

// make sure to implement interfaces correctly
var _ admindb.AllowListService = (*AllowList)(nil)

type AllowList struct {
	db *sql.DB
}

// Add adds the feed to the list.
func (l AllowList) Add(ctx context.Context, a refs.FeedRef) error {
	// TODO: better valid
	if _, err := refs.ParseFeedRef(a.Ref()); err != nil {
		return err
	}

	var entry models.AllowList

	entry.PubKey.FeedRef = a
	err := entry.Insert(ctx, l.db, boil.Whitelist("pub_key"))
	if err != nil {
		return fmt.Errorf("allow-list: failed to insert new entry %s: %w", entry.PubKey, err)
	}

	return nil
}

// Has returns true if a feed is on the list.
func (l AllowList) Has(ctx context.Context, h refs.FeedRef) bool {
	_, err := models.AllowLists(qm.Where("pub_key = ?", h.Ref())).One(ctx, l.db)
	if err != nil {
		return false
	}
	return true
}

// List returns a list of all the feeds.
func (l AllowList) List(ctx context.Context) ([]refs.FeedRef, error) {
	all, err := models.AllowLists().All(ctx, l.db)
	if err != nil {
		return nil, err
	}

	var asRefs = make([]refs.FeedRef, len(all))
	for i, allowed := range all {
		fmt.Println(allowed.PubKey.Ref())

		asRefs[i] = allowed.PubKey.FeedRef
	}

	return asRefs, nil
}

// Remove removes the feed from the list.
func (l AllowList) Remove(ctx context.Context, r refs.FeedRef) error {
	entry, err := models.AllowLists(qm.Where("pub_key = ?", r.Ref())).One(ctx, l.db)
	if err != nil {
		return err
	}

	_, err = entry.Delete(ctx, l.db)
	if err != nil {
		return err
	}

	return nil
}
