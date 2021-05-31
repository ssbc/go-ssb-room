package sqlite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb/sqlite/models"
)

func TestMembers(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	// broken feed (unknown algo)
	tf := refs.FeedRef{ID: bytes.Repeat([]byte("fooo"), 8), Algo: "nope"}
	_, err = db.Members.Add(ctx, tf, roomdb.RoleMember)
	r.Error(err)

	// looks ok at least
	okFeed := refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: refs.RefAlgoFeedSSB1}
	mid, err := db.Members.Add(ctx, okFeed, roomdb.RoleMember)
	r.NoError(err)

	sqlDB := db.Members.db
	count, err := models.Members().Count(ctx, sqlDB)
	r.NoError(err)
	r.EqualValues(count, 1)

	lst, err := db.Members.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	_, yes := db.Members.GetByFeed(ctx, okFeed)
	r.NoError(yes)

	okMember, err := db.Members.GetByFeed(ctx, okFeed)
	r.NoError(err)
	r.Equal(okMember.ID, mid)
	r.Equal(okMember.Role, roomdb.RoleMember)
	r.True(okMember.PubKey.Equal(&okFeed))

	_, yes = db.Members.GetByFeed(ctx, tf)
	r.Error(yes)

	err = db.Members.RemoveFeed(ctx, okFeed)
	r.NoError(err)

	count, err = models.Members().Count(ctx, sqlDB)
	r.NoError(err)
	r.EqualValues(count, 0)

	lst, err = db.Members.List(ctx)
	r.NoError(err)
	r.Len(lst, 0)

	_, yes = db.Members.GetByFeed(ctx, okFeed)
	r.Error(yes)

	r.NoError(db.Close())
}

func TestMembersUnique(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: refs.RefAlgoFeedSSB1}
	_, err = db.Members.Add(ctx, feedA, roomdb.RoleMember)
	r.NoError(err)

	_, err = db.Members.Add(ctx, feedA, roomdb.RoleMember)
	r.Error(err)

	lst, err := db.Members.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	r.NoError(db.Close())
}

func TestMembersByID(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: refs.RefAlgoFeedSSB1}
	_, err = db.Members.Add(ctx, feedA, roomdb.RoleMember)
	r.NoError(err)

	lst, err := db.Members.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	_, yes := db.Members.GetByID(ctx, lst[0].ID)
	r.NoError(yes)

	_, yes = db.Members.GetByID(ctx, 666)
	r.Error(yes)

	err = db.Members.RemoveID(ctx, 666)
	r.Error(err)
	r.EqualError(err, roomdb.ErrNotFound.Error())

	err = db.Members.RemoveID(ctx, lst[0].ID)
	r.NoError(err)

	_, yes = db.Members.GetByID(ctx, lst[0].ID)
	r.Error(yes)

	r.NoError(db.Close())
}

func TestMembersSetRole(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	// create two users
	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("1"), 32), Algo: refs.RefAlgoFeedSSB1}
	idA, err := db.Members.Add(ctx, feedA, roomdb.RoleAdmin)
	r.NoError(err)
	t.Log("member A:", idA)

	feedB := refs.FeedRef{ID: bytes.Repeat([]byte("2"), 32), Algo: refs.RefAlgoFeedSSB1}
	idB, err := db.Members.Add(ctx, feedB, roomdb.RoleModerator)
	r.NoError(err)
	t.Log("member B:", idB)

	// list and check
	members, err := db.Members.List(ctx)
	r.NoError(err)
	r.Len(members, 2)
	findMemberWithRole(t, members, idA, roomdb.RoleAdmin)
	findMemberWithRole(t, members, idB, roomdb.RoleModerator)

	// upgrade B to admin
	err = db.Members.SetRole(ctx, idB, roomdb.RoleAdmin)
	r.NoError(err)

	// list and check
	members, err = db.Members.List(ctx)
	r.NoError(err)
	r.Len(members, 2)
	findMemberWithRole(t, members, idA, roomdb.RoleAdmin)
	findMemberWithRole(t, members, idB, roomdb.RoleAdmin)

	// downgrade A to member
	err = db.Members.SetRole(ctx, idA, roomdb.RoleMember)
	r.NoError(err)

	// list and check
	members, err = db.Members.List(ctx)
	r.NoError(err)
	r.Len(members, 2)
	findMemberWithRole(t, members, idA, roomdb.RoleMember)
	findMemberWithRole(t, members, idB, roomdb.RoleAdmin)

	// can't downgrade B to member (need one admin)
	err = db.Members.SetRole(ctx, idB, roomdb.RoleMember)
	r.Error(err)

	// unchanged
	members, err = db.Members.List(ctx)
	r.NoError(err)
	r.Len(members, 2)
	findMemberWithRole(t, members, idA, roomdb.RoleMember)
	findMemberWithRole(t, members, idB, roomdb.RoleAdmin)

	r.NoError(db.Close())
}

func findMemberWithRole(t *testing.T, members []roomdb.Member, id int64, r roomdb.Role) {
	var found = false

	for _, m := range members {
		if m.ID == id {
			if m.Role != r {
				t.Errorf("member %d has the wrong role (has %s)", m.ID, m.Role)
			}
			found = true
		}
	}
	if !found {
		t.Errorf("member %d not in the list", id)
	}
}

func TestMembersAliases(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: refs.RefAlgoFeedSSB1}
	mid, err := db.Members.Add(ctx, feedA, roomdb.RoleMember)
	r.NoError(err)

	lst, err := db.Members.List(ctx)
	r.NoError(err)
	r.Len(lst, 1)

	err = db.Aliases.Register(ctx, "foo", feedA, []byte("just-a-test"))
	r.NoError(err)

	err = db.Aliases.Register(ctx, "bar", feedA, []byte("just-a-test-two"))
	r.NoError(err)

	storedMember, err := db.Members.GetByID(ctx, mid)
	r.NoError(err)
	r.Len(storedMember.Aliases, 2)

	storedMember, err = db.Members.GetByFeed(ctx, feedA)
	r.NoError(err)
	r.Len(storedMember.Aliases, 2)
}
