// SPDX-License-Identifier: MIT

package sqlite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
	"github.com/stretchr/testify/require"
	refs "go.mindeco.de/ssb-refs"
)

func TestDeniedKeys(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	tf := refs.FeedRef{ID: bytes.Repeat([]byte("fooo"), 8), Algo: "nope"}
	err = db.DeniedKeys.Add(ctx, tf, "wont work anyhow")
	r.Error(err)

	// looks ok at least
	created := time.Now()
	time.Sleep(time.Second / 2)
	okFeed := refs.FeedRef{ID: bytes.Repeat([]byte("b44d"), 8), Algo: refs.RefAlgoFeedSSB1}
	err = db.DeniedKeys.Add(ctx, okFeed, "be gone")
	r.NoError(err)

	// hack into the interface to get the concrete database/sql instance
	sqlDB := db.DeniedKeys.db

	count, err := models.DeniedKeys().Count(ctx, sqlDB)
	r.NoError(err)
	r.EqualValues(count, 1)

	lst, err := db.DeniedKeys.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)
	r.Equal(okFeed.Ref(), lst[0].PubKey.Ref())
	r.Equal("be gone", lst[0].Comment)
	r.True(lst[0].CreatedAt.After(created))

	yes := db.DeniedKeys.HasFeed(ctx, okFeed)
	r.True(yes)

	yes = db.DeniedKeys.HasFeed(ctx, tf)
	r.False(yes)

	err = db.DeniedKeys.RemoveFeed(ctx, okFeed)
	r.NoError(err)

	count, err = models.DeniedKeys().Count(ctx, sqlDB)
	r.NoError(err)
	r.EqualValues(count, 0)

	lst, err = db.DeniedKeys.List(ctx)
	r.NoError(err)
	r.Len(lst, 0)

	yes = db.DeniedKeys.HasFeed(ctx, okFeed)
	r.False(yes)

	r.NoError(db.Close())
}

func TestDeniedKeysUnique(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("b33f"), 8), Algo: refs.RefAlgoFeedSSB1}
	err = db.DeniedKeys.Add(ctx, feedA, "test comment")
	r.NoError(err)

	err = db.DeniedKeys.Add(ctx, feedA, "test comment")
	r.Error(err)

	lst, err := db.DeniedKeys.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	r.NoError(db.Close())
}

func TestDeniedKeysByID(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("b33f"), 8), Algo: refs.RefAlgoFeedSSB1}
	err = db.DeniedKeys.Add(ctx, feedA, "nope")
	r.NoError(err)

	lst, err := db.DeniedKeys.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	yes := db.DeniedKeys.HasID(ctx, lst[0].ID)
	r.True(yes)

	yes = db.DeniedKeys.HasID(ctx, 666)
	r.False(yes)

	err = db.DeniedKeys.RemoveID(ctx, 666)
	r.Error(err)
	r.EqualError(err, roomdb.ErrNotFound.Error())

	err = db.DeniedKeys.RemoveID(ctx, lst[0].ID)
	r.NoError(err)

	yes = db.DeniedKeys.HasID(ctx, lst[0].ID)
	r.False(yes)
}
