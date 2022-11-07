// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package sqlite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ssbc/go-ssb-room/v2/roomdb"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/repo"
	"github.com/ssbc/go-ssb-room/v2/roomdb/sqlite/models"
	"github.com/stretchr/testify/require"
)

func TestDeniedKeys(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	tf, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("fooo"), 8), "nope")
	if err != nil {
		r.Error(err)
	}
	err = db.DeniedKeys.Add(ctx, tf, "wont work anyhow")
	r.Error(err)

	// looks ok at least
	created := time.Now()
	time.Sleep(time.Second)
	okFeed, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("b44d"), 8), refs.RefAlgoFeedSSB1)
	if err != nil {
		r.Error(err)
	}
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
	r.Equal(okFeed.String(), lst[0].PubKey.String())
	r.Equal("be gone", lst[0].Comment)
	r.True(lst[0].CreatedAt.After(created), "not created after the sleep?")

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

	feedA, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("b33f"), 8), refs.RefAlgoFeedSSB1)
	if err != nil {
		r.Error(err)
	}
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

	feedA, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("b33f"), 8), refs.RefAlgoFeedSSB1)
	if err != nil {
		r.Error(err)
	}
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
