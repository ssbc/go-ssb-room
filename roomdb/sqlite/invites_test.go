package sqlite

import (
	"bytes"
	"context"
	"encoding/base64"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
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

		_, err := db.Invites.Create(ctx, 666)
		r.Error(err, "can't create invite for invalid user")
	})

	invitingMember := refs.FeedRef{ID: bytes.Repeat([]byte("ohai"), 8), Algo: refs.RefAlgoFeedSSB1}
	mid, err := db.Members.Add(ctx, invitingMember, roomdb.RoleModerator)
	require.NoError(t, err, "failed to create test user")

	t.Run("simple create and consume", func(t *testing.T) {
		r := require.New(t)

		// i really don't want to do a mocked time functions and rather solve the comment in migration 6 instead
		before := time.Now()

		tok, err := db.Invites.Create(ctx, mid)
		r.NoError(err, "failed to create invite token")

		_, err = base64.URLEncoding.DecodeString(tok)
		r.NoError(err, "not a valid base64 string")

		lst, err := db.Invites.List(ctx)
		r.NoError(err, "failed to get list of tokens")
		r.Len(lst, 1, "expected 1 invite")

		r.True(lst[0].CreatedAt.After(before), "expected CreatedAt to be after the start marker")

		_, nope := db.Members.GetByFeed(ctx, newMember)
		r.Error(nope, "expected feed to not yet be on the allow list")

		gotInv, err := db.Invites.GetByToken(ctx, tok)
		r.NoError(err)
		r.Equal(lst[0].ID, gotInv.ID)

		inv, err := db.Invites.Consume(ctx, tok, newMember)
		r.NoError(err, "failed to consume the invite")
		r.NotEqualValues(0, inv.ID, "invite ID unset")
		r.True(inv.CreatedAt.After(before), "expected CreatedAt to be after the start marker")

		// consume also adds it to the allow list
		_, yes := db.Members.GetByFeed(ctx, newMember)
		r.NoError(yes, "expected feed on the allow list")

		lst, err = db.Invites.List(ctx)
		r.NoError(err, "failed to get list of tokens post consume")
		r.Len(lst, 0, "expected no active invites")

		// can't use twice
		_, err = db.Invites.Consume(ctx, tok, newMember)
		r.Error(err, "failed to consume the invite")

	})

	t.Run("simple create but revoke before use", func(t *testing.T) {
		r := require.New(t)

		tok, err := db.Invites.Create(ctx, mid)
		r.NoError(err, "failed to create invite token")

		lst, err := db.Invites.List(ctx)
		r.NoError(err, "failed to get list of tokens")
		r.Len(lst, 1, "expected 1 invite")

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
