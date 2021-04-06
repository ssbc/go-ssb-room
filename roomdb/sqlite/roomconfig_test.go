// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/stretchr/testify/require"
)

func TestRoomConfig(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)

	tr := repo.New(testRepo)

	db, err := Open(tr)
	r.NoError(err)

	// test setting a valid privacy mode
	err = db.Config.SetPrivacyMode(ctx, roomdb.ModeCommunity)
	r.NoError(err)

	// make sure the mode was set correctly by getting it
	pm, err := db.Config.GetPrivacyMode(ctx)
	r.NoError(err)
	r.Equal(pm, roomdb.ModeCommunity, "privacy mode was unknown")

	// test setting an invalid privacy mode
	err = db.Config.SetPrivacyMode(ctx, 1337)
	r.Error(err)
}
