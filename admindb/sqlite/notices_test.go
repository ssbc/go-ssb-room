package sqlite

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/stretchr/testify/require"
)

func TestNoticesCRUD(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	r.NoError(err)

	// boil.DebugWriter = os.Stderr
	// boil.DebugMode = true

	t.Run("not found", func(t *testing.T) {
		r := require.New(t)

		_, err = db.Notices.GetByID(ctx, 9999)
		r.Error(err)
		r.EqualError(err, admindb.ErrNotFound.Error())

		err = db.Notices.RemoveID(ctx, 9999)
		r.Error(err)
		r.EqualError(err, admindb.ErrNotFound.Error())
	})

	t.Run("new and update", func(t *testing.T) {
		r := require.New(t)
		var n admindb.Notice

		n.Title = fmt.Sprintf("Test notice %d", rand.Int())
		n.Content = `# This is **not** a test!`
		n.Language = "en-GB"

		err := db.Notices.Save(ctx, &n)
		r.NoError(err, "failed to save")
		r.NotEqual(0, n.ID, "should have a fresh id")

		got, err := db.Notices.GetByID(ctx, n.ID)
		r.NoError(err, "failed to get saved entry")
		r.Equal(n.Title, got.Title)
		r.Equal(n.ID, got.ID)
		r.Equal(n.Language, got.Language)

		oldID := n.ID
		n.Title = fmt.Sprintf("Updated test notice %d", rand.Int())
		err = db.Notices.Save(ctx, &n)
		r.NoError(err, "failed to save")
		r.Equal(oldID, n.ID, "should have the same ID")

		// be gone
		err = db.Notices.RemoveID(ctx, oldID)
		r.NoError(err)

		_, err = db.Notices.GetByID(ctx, oldID)
		r.Error(err)
		r.EqualError(err, admindb.ErrNotFound.Error())
	})
}

func TestPinnedNotices(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	r.NoError(err)

	t.Run("defaults", func(t *testing.T) {
		allTheNotices, err := db.PinnedNotices.List(ctx)
		r.NoError(err)

		type expectedNotices struct {
			Name  admindb.PinnedNoticeName
			Count int
		}

		cases := []expectedNotices{
			{admindb.NoticeDescription, 2},
			{admindb.NoticeNews, 1},
			{admindb.NoticePrivacyPolicy, 2},
			{admindb.NoticeCodeOfConduct, 1},
		}

		for i, tcase := range cases {
			desc, has := allTheNotices[tcase.Name]
			r.True(has, "case %d failed - notice %s not in map", i, tcase.Name)
			r.Len(desc, tcase.Count, "case %d failed - wrong number of notices for %s", i, tcase.Name)
		}
	})

	t.Run("validity", func(t *testing.T) {
		var empty admindb.Notice
		// no id
		err = db.PinnedNotices.Set(ctx, admindb.NoticeNews, empty.ID)
		r.Error(err)

		// not-null id
		empty.ID = 999
		err = db.PinnedNotices.Set(ctx, admindb.NoticeNews, empty.ID)
		r.Error(err)

		// invalid notice name
		err = db.PinnedNotices.Set(ctx, "unknown", empty.ID)
		r.Error(err)
	})

	t.Run("add new localization", func(t *testing.T) {
		var notice admindb.Notice
		notice.Title = "política de privacidad"
		notice.Content = "solo una prueba"
		notice.Language = "es"

		// save the new notice
		err = db.Notices.Save(ctx, &notice)
		r.NoError(err)

		// set it
		err = db.PinnedNotices.Set(ctx, admindb.NoticePrivacyPolicy, notice.ID)
		r.NoError(err)

		// retreive it
		ret, err := db.PinnedNotices.Get(ctx, admindb.NoticePrivacyPolicy, notice.Language)
		r.NoError(err)
		r.Equal(notice, *ret, "notices are not the same")

		// see that it's in the list
		allTheNotices, err := db.PinnedNotices.List(ctx)
		r.NoError(err)

		notices, has := allTheNotices[admindb.NoticePrivacyPolicy]
		r.True(has)
		r.Len(notices, 3)

		has = false
		for _, n := range notices {
			if n.Title == notice.Title {
				has = true
				break
			}
		}
		r.True(has, "did not find new notice in list()")
	})

}
