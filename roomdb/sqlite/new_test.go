package sqlite

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	refs "go.mindeco.de/ssb-refs"
)

// verify the database opens and migrates successfully from zero state
func TestSchema(t *testing.T) {
	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}

func TestBasic(t *testing.T) {
	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	ctx := context.Background()
	feedA := refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: refs.RefAlgoFeedSSB1}
	memberID, err := db.Members.Add(ctx, feedA, roomdb.RoleMember)
	require.NoError(t, err)
	require.NotEqual(t, 0, memberID)

	err = db.AuthFallback.Create(ctx, memberID, "testLogin", []byte("super-cheesy-password-12345"))
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}
