package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/volatiletech/sqlboiler/v4/boil"

	"github.com/friendsofgo/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb/sqlite/models"
)

// make sure to implement interfaces correctly
var _ admindb.PinnedNoticesService = (*PinnedNotices)(nil)

type PinnedNotices struct {
	db *sql.DB
}

func (pndb PinnedNotices) Set(name admindb.PinnedNoticeName, pageID int64, lang string) error {
	if !name.Valid() {
		return fmt.Errorf("fixed pages: invalid page name: %s", name)
	}

	return fmt.Errorf("TODO: set fixed page %s to %d:%s", name, pageID, lang)
}

// make sure to implement interfaces correctly
var _ admindb.NoticesService = (*Notices)(nil)

type Notices struct {
	db *sql.DB
}

func (ndb Notices) GetByID(ctx context.Context, id int64) (admindb.Notice, error) {
	var n admindb.Notice

	dbEntry, err := models.FindNotice(ctx, ndb.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return n, admindb.ErrNotFound
		}
		return n, err
	}

	// convert models type to admindb type
	n.ID = dbEntry.ID
	n.Title = dbEntry.Title
	n.Language = dbEntry.Language
	n.Content = dbEntry.Content

	return n, nil
}

func (ndb Notices) RemoveID(ctx context.Context, id int64) error {
	dbEntry, err := models.FindNotice(ctx, ndb.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admindb.ErrNotFound
		}
		return err
	}

	_, err = dbEntry.Delete(ctx, ndb.db)
	if err != nil {
		return err
	}

	return nil
}

func (ndb Notices) Save(ctx context.Context, p *admindb.Notice) error {
	if p.ID == 0 {
		var newEntry models.Notice
		newEntry.Title = p.Title
		newEntry.Content = p.Content
		newEntry.Language = p.Language
		err := newEntry.Insert(ctx, ndb.db, boil.Whitelist("title", "content", "language"))
		if err != nil {
			return err
		}
		p.ID = newEntry.ID
		return nil
	}

	var existing models.Notice
	existing.ID = p.ID
	existing.Title = p.Title
	existing.Content = p.Content
	existing.Language = p.Language
	_, err := existing.Update(ctx, ndb.db, boil.Whitelist("title", "content", "language"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return admindb.ErrNotFound
		}
		return err
	}

	return nil
}
