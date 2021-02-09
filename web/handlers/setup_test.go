// SPDX-License-Identifier: MIT

package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"go.mindeco.de/http/tester"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

var (
	testMux    *http.ServeMux
	testClient *tester.Tester
	testRouter = router.CompleteApp()

	// mocked dbs
	testAuthDB         *mockdb.FakeAuthWithSSBService
	testAuthFallbackDB *mockdb.FakeAuthFallbackService
)

func setup(t *testing.T) {

	testRepoPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepoPath)
	testRepo := repo.New(testRepoPath)

	testAuthDB = new(mockdb.FakeAuthWithSSBService)
	testAuthFallbackDB = new(mockdb.FakeAuthFallbackService)
	h, err := New(
		testRouter,
		testRepo,
		testAuthDB,
		testAuthFallbackDB,
	)
	if err != nil {
		t.Fatal(errors.Wrap(err, "setup: handler init failed"))
	}

	// log, _ := logtest.KitLogger("complete", t)

	testMux = http.NewServeMux()
	testMux.Handle("/", h)
	testClient = tester.New(testMux, t)
}

func teardown() {
	testMux = nil
	testClient = nil
	testAuthDB = nil
	testAuthFallbackDB = nil
}
