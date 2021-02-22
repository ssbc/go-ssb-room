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
