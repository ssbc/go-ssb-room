package sqlite

import (
	"bytes"
	"context"
	"encoding/base64"
	"math/rand"
	"os"
	"path/filepath"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/stretchr/testify/require"
	refs "go.mindeco.de/ssb-refs"
)

func TestInvites(t *testing.T) {
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)
	tr := repo.New(testRepo)

	// fake feed for testing, looks ok at least
	newMember := refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: refs.RefAlgoFeedSSB1}

	db, err := Open(tr)
	require.NoError(t, err)

	t.Run("try to consume invalid token", func(t *testing.T) {
		r := require.New(t)

		lst, err := db.Invites.List(ctx)
		r.NoError(err, "failed to get empty list of tokens")
		r.Len(lst, 0, "expected no active invites")

		randToken := make([]byte, 32)
		rand.Read(randToken)

		_, err = db.Invites.Consume(ctx, string(randToken), newMember)
		r.Error(err, "expected error for inactive invite")
	})

	t.Run("user needs to exist", func(t *testing.T) {
		r := require.New(t)

		_, err := db.Invites.Create(ctx, 666, "")
		r.Error(err, "can't create invite for invalid user")
	})

	testUserName := "test-user"
	uid, err := db.AuthFallback.Create(ctx, testUserName, []byte("bad-password"))
	require.NoError(t, err, "failed to create test user")

	t.Run("simple create and consume", func(t *testing.T) {
		r := require.New(t)

		tok, err := db.Invites.Create(ctx, uid, "bestie")
		r.NoError(err, "failed to create invite token")

		_, err = base64.URLEncoding.DecodeString(tok)
		r.NoError(err, "not a valid base64 string")

		lst, err := db.Invites.List(ctx)
		r.NoError(err, "failed to get list of tokens")
		r.Len(lst, 1, "expected 1 invite")

		r.Equal("bestie", lst[0].AliasSuggestion)
		r.Equal(testUserName, lst[0].CreatedBy.Name)

		inv, err := db.Invites.Consume(ctx, tok, newMember)
		r.NoError(err, "failed to consume the invite")
		r.Equal(testUserName, inv.CreatedBy.Name)
		r.NotEqualValues(0, inv.ID, "invite ID unset")

		lst, err = db.Invites.List(ctx)
		r.NoError(err, "failed to get list of tokens post consume")
		r.Len(lst, 0, "expected no active invites")

		// can't use twice
		_, err = db.Invites.Consume(ctx, tok, newMember)
		r.Error(err, "failed to consume the invite")
	})

	t.Run("simple create but revoke before use", func(t *testing.T) {
		r := require.New(t)

		tok, err := db.Invites.Create(ctx, uid, "bestie")
		r.NoError(err, "failed to create invite token")

		lst, err := db.Invites.List(ctx)
		r.NoError(err, "failed to get list of tokens")
		r.Len(lst, 1, "expected 1 invite")

		r.Equal("bestie", lst[0].AliasSuggestion)
		r.Equal(testUserName, lst[0].CreatedBy.Name)

		err = db.Invites.Revoke(ctx, lst[0].ID)
		r.NoError(err, "failed to consume the invite")

		lst, err = db.Invites.List(ctx)
		r.NoError(err, "failed to get list of tokens post consume")
		r.Len(lst, 0, "expected no active invites")

		// can't use twice
		_, err = db.Invites.Consume(ctx, tok, newMember)
		r.Error(err, "failed to consume the invite")
	})

}
