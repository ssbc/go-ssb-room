// SPDX-License-Identifier: MIT

package sqlite

import (
	"bytes"
	"context"
	"crypto/rand"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	refs "go.mindeco.de/ssb-refs"
)

func TestAliases(t *testing.T) {
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)
	tr := repo.New(testRepo)

	// fake feed for testing, looks ok at least
	newMember := refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: refs.RefAlgoFeedSSB1}

	// 64 bytes of random for testing (validation is handled by the handlers)
	testSig := make([]byte, 64)
	rand.Read(testSig)

	db, err := Open(tr)
	require.NoError(t, err)

	t.Run("not found", func(t *testing.T) {
		r := require.New(t)

		lst, err := db.Aliases.List(ctx)
		r.NoError(err)
		r.Len(lst, 0)

		_, err = db.Aliases.GetByID(ctx, 9999)
		r.Error(err)
		r.EqualError(err, roomdb.ErrNotFound.Error())

		_, err = db.Aliases.Resolve(ctx, "unknown")
		r.Error(err)
		r.EqualError(err, roomdb.ErrNotFound.Error())

		err = db.Aliases.Revoke(ctx, "unknown")
		r.Error(err)
		r.EqualError(errors.Unwrap(err), roomdb.ErrNotFound.Error())
	})

	t.Run("register and revoke again", func(t *testing.T) {
		r := require.New(t)

		testName := "flaky"

		// shouldnt work while not a member
		err = db.Aliases.Register(ctx, testName, newMember, testSig)
		r.Error(err)

		// allow the member
		_, err = db.Members.Add(ctx, newMember, roomdb.RoleMember)
		r.NoError(err)

		err = db.Aliases.Register(ctx, testName, newMember, testSig)
		r.NoError(err)

		// should have one alias now
		lst, err := db.Aliases.List(ctx)
		r.NoError(err)
		r.Len(lst, 1)

		aliasByID, err := db.Aliases.GetByID(ctx, lst[0].ID)
		r.NoError(err)
		r.Equal(testName, aliasByID.Name)
		r.Equal(testSig, aliasByID.Signature)

		resolvedAlias, err := db.Aliases.Resolve(ctx, testName)
		r.NoError(err)
		r.Equal(aliasByID, resolvedAlias)

		err = db.Aliases.Revoke(ctx, testName)
		r.NoError(err)

		_, err = db.Aliases.GetByID(ctx, lst[0].ID)
		r.Error(err)
		r.EqualError(err, roomdb.ErrNotFound.Error())

		_, err = db.Aliases.Resolve(ctx, testName)
		r.Error(err)
		r.EqualError(err, roomdb.ErrNotFound.Error())
	})
}

func TestAliasesUniqueError(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)
	tr := repo.New(testRepo)

	db, err := Open(tr)
	r.NoError(err)

	// fake feed for testing, looks ok at least
	newMember := refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: refs.RefAlgoFeedSSB1}

	// 64 bytes of random for testing (validation is handled by the handlers)
	testSig := make([]byte, 64)
	rand.Read(testSig)

	testName := "thealias"

	// allow the member
	_, err = db.Members.Add(ctx, newMember, roomdb.RoleMember)
	r.NoError(err)

	err = db.Aliases.Register(ctx, testName, newMember, testSig)
	r.NoError(err)

	// should have one alias now
	lst, err := db.Aliases.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	err = db.Aliases.Register(ctx, testName, newMember, testSig)
	r.Error(err)
	var takenErr roomdb.ErrAliasTaken
	r.True(errors.As(err, &takenErr), "expected a special error value")
	r.Equal(testName, takenErr.Name)
}
