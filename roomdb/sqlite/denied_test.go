// +build ignore

package sqlite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
	"github.com/stretchr/testify/require"
	refs "go.mindeco.de/ssb-refs"
)

func TestDeniedList(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	tf := refs.FeedRef{ID: bytes.Repeat([]byte("fooo"), 8), Algo: "nope"}
	err = db.DeniedList.Add(ctx, tf)
	r.Error(err)

	// looks ok at least
	okFeed := refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: refs.RefAlgoFeedSSB1}
	err = db.DeniedList.Add(ctx, okFeed)
	r.NoError(err)

	// hack into the interface to get the concrete database/sql instance
	sqlDB := db.DeniedList.(*DeniedList).db

	count, err := models.DeniedLists().Count(ctx, sqlDB)
	r.NoError(err)
	r.EqualValues(count, 1)

	lst, err := db.DeniedList.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	yes := db.DeniedList.HasFeed(ctx, okFeed)
	r.True(yes)

	yes = db.DeniedList.HasFeed(ctx, tf)
	r.False(yes)

	err = db.DeniedList.RemoveFeed(ctx, okFeed)
	r.NoError(err)

	count, err = models.DeniedLists().Count(ctx, sqlDB)
	r.NoError(err)
	r.EqualValues(count, 0)

	lst, err = db.DeniedList.List(ctx)
	r.NoError(err)
	r.Len(lst, 0)

	yes = db.DeniedList.HasFeed(ctx, okFeed)
	r.False(yes)

	r.NoError(db.Close())
}

func TestDeniedListUnique(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: refs.RefAlgoFeedSSB1}
	err = db.DeniedList.Add(ctx, feedA)
	r.NoError(err)

	err = db.DeniedList.Add(ctx, feedA)
	r.Error(err)

	lst, err := db.DeniedList.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	r.NoError(db.Close())
}

func TestDeniedListByID(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: refs.RefAlgoFeedSSB1}
	err = db.DeniedList.Add(ctx, feedA)
	r.NoError(err)

	lst, err := db.DeniedList.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	yes := db.DeniedList.HasID(ctx, lst[0].ID)
	r.True(yes)

	yes = db.DeniedList.HasID(ctx, 666)
	r.False(yes)

	err = db.DeniedList.RemoveID(ctx, 666)
	r.Error(err)
	r.EqualError(err, roomdb.ErrNotFound.Error())

	err = db.DeniedList.RemoveID(ctx, lst[0].ID)
	r.NoError(err)

	yes = db.DeniedList.HasID(ctx, lst[0].ID)
	r.False(yes)
}
