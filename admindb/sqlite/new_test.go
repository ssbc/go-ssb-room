package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	_ "github.com/mattn/go-sqlite3"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/repo"
	"github.com/stretchr/testify/require"
)

// verify the database opens and migrates successfully from zero state
func TestSimple(t *testing.T) {
	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	require.NoError(t, err)

	ctx := context.Background()
	err = db.AuthFallback.Create(ctx, "testUser", []byte("super-cheesy-password-12345"))
	require.NoError(t, err)

	err = db.Close()
	require.NoError(t, err)
}
