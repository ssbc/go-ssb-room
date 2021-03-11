package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.PinnedNoticesService = (*PinnedNotices)(nil)

type PinnedNotices struct {
	db *sql.DB
}

// List returns a sortable map of all the pinned notices
func (pn PinnedNotices) List(ctx context.Context) (roomdb.PinnedNotices, error) {

	// get all the pins and eager-load the related notices
	lst, err := models.Pins(qm.Load("Notices")).All(ctx, pn.db)
	if err != nil {
		return nil, err
	}

	// prepare a map for them
	var pinnedMap = make(roomdb.PinnedNotices, len(lst))

	for _, entry := range lst {

		// skip a pin if it has no entries
		noticeCount := len(entry.R.Notices)
		if noticeCount == 0 {
			continue
		}

		name := roomdb.PinnedNoticeName(entry.Name)

		// get the exisint map entry or create a new slice
		notices, has := pinnedMap[name]
		if !has {
			notices = make([]roomdb.Notice, 0, noticeCount)
		}

		// add all the related notice to the slice
		for _, n := range entry.R.Notices {
			relatedNotice := roomdb.Notice{
				ID:       n.ID,
				Title:    n.Title,
				Content:  n.Content,
				Language: n.Language,
			}
			notices = append(notices, relatedNotice)
		}

		// update the map
		pinnedMap[name] = notices
	}

	return pinnedMap, nil
}

func (pn PinnedNotices) Get(ctx context.Context, name roomdb.PinnedNoticeName, lang string) (*roomdb.Notice, error) {
	p, err := models.Pins(
		qm.Where("name = ?", name),
		qm.Load("Notices", qm.Where("language = ?", lang)),
	).One(ctx, pn.db)
	if err != nil {
		return nil, err
	}

	if n := len(p.R.Notices); n != 1 {
		return nil, fmt.Errorf("pinnedNotice: expected 1 notice but got %d", n)
	}

	modelNotice := p.R.Notices[0]

	return &roomdb.Notice{
		ID:       modelNotice.ID,
		Title:    modelNotice.Title,
		Content:  modelNotice.Content,
		Language: modelNotice.Language,
	}, nil
}

func (pn PinnedNotices) Set(ctx context.Context, name roomdb.PinnedNoticeName, noticeID int64) error {
	if !name.Valid() {
		return fmt.Errorf("pinned notice: invalid notice name: %s", name)
	}

	n, err := models.FindNotice(ctx, pn.db, noticeID)
	if err != nil {
		return err
	}

	p, err := models.Pins(qm.Where("name = ?", name)).One(ctx, pn.db)
	if err != nil {
		return err
	}

	err = p.AddNotices(ctx, pn.db, false, n)
	if err != nil {
		return err
	}

	return nil
}

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.NoticesService = (*Notices)(nil)

type Notices struct {
	db *sql.DB
}

func (n Notices) GetByID(ctx context.Context, id int64) (roomdb.Notice, error) {
	var notice roomdb.Notice

	dbEntry, err := models.FindNotice(ctx, n.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return notice, roomdb.ErrNotFound
		}
		return notice, err
	}

	// convert models type to admindb type
	notice.ID = dbEntry.ID
	notice.Title = dbEntry.Title
	notice.Language = dbEntry.Language
	notice.Content = dbEntry.Content

	return notice, nil
}

func (n Notices) RemoveID(ctx context.Context, id int64) error {
	dbEntry, err := models.FindNotice(ctx, n.db, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return roomdb.ErrNotFound
		}
		return err
	}

	_, err = dbEntry.Delete(ctx, n.db)
	if err != nil {
		return err
	}

	return nil
}

func (n Notices) Save(ctx context.Context, p *roomdb.Notice) error {
	if p.ID == 0 {
		var newEntry models.Notice
		newEntry.Title = p.Title
		newEntry.Content = p.Content
		newEntry.Language = p.Language
		err := newEntry.Insert(ctx, n.db, boil.Whitelist("title", "content", "language"))
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
	_, err := existing.Update(ctx, n.db, boil.Whitelist("title", "content", "language"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return roomdb.ErrNotFound
		}
		return err
	}

	return nil
}
