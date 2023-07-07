// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package sqlite

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	_ "modernc.org/sqlite"

	"github.com/ssbc/go-ssb-room/v2/internal/repo"
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
