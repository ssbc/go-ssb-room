package handlers

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/pkg/errors"
	"go.mindeco.de/http/tester"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb/mockdb"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/repo"
	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
)

var (
	testMux    *http.ServeMux
	testClient *tester.Tester
	testRouter = router.CompleteApp()

	// mocked dbs
	testAuthDB         *mockdb.FakeAuthService
	testAuthFallbackDB *mockdb.FakeFallbackAuth
)

func setup(t *testing.T) {

	testRepoPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepoPath)
	testRepo := repo.New(testRepoPath)

	testAuthDB = new(mockdb.FakeAuthService)
	testAuthFallbackDB = new(mockdb.FakeFallbackAuth)
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
	testAuthFallbackDB = nil
	testAuthFallbackDB = nil
}
