// SPDX-License-Identifier: MIT

package handlers

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/pkg/errors"
	"go.mindeco.de/http/tester"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

var (
	testMux    *http.ServeMux
	testClient *tester.Tester
	testRouter = router.CompleteApp()

	// mocked dbs
	testAuthDB         *mockdb.FakeAuthWithSSBService
	testAuthFallbackDB *mockdb.FakeAuthFallbackService

	testI18N = justTheKeys()
)

func setup(t *testing.T) {

	testRepoPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepoPath)
	testRepo := repo.New(testRepoPath)

	testOverride := filepath.Join(testRepo.GetPath("i18n"), "active.en.toml")
	os.MkdirAll(filepath.Dir(testOverride), 0700)
	err := ioutil.WriteFile(testOverride, testI18N, 0700)
	if err != nil {
		t.Fatal(err)
	}

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

// auto generate from defaults a list of Label = "Label"
func justTheKeys() []byte {
	f, err := i18n.Defaults.Open("/active.en.toml")
	if err != nil {
		panic(err)
	}
	justAMap := make(map[string]interface{})
	md, err := toml.DecodeReader(f, &justAMap)
	if err != nil {
		panic(err)
	}

	var buf = &bytes.Buffer{}

	for _, key := range md.Keys() {
		fmt.Fprintf(buf, "%s = \"%s\"\n", key, key)
	}

	return buf.Bytes()
}
