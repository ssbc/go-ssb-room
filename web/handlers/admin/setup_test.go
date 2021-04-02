// SPDX-License-Identifier: MIT

package admin

import (
	"context"
	"crypto/rand"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"go.mindeco.de/http/render"
	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/randutil"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	weberrs "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n/i18ntesting"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

type testSession struct {
	Domain string
	Mux    *http.ServeMux

	Client *tester.Tester

	Router *mux.Router
	URLTo  web.URLMaker

	AliasesDB    *mockdb.FakeAliasesService
	DeniedKeysDB *mockdb.FakeDeniedKeysService
	InvitesDB    *mockdb.FakeInvitesService
	NoticeDB     *mockdb.FakeNoticesService
	MembersDB    *mockdb.FakeMembersService
	PinnedDB     *mockdb.FakePinnedNoticesService

	User roomdb.Member

	RoomState *roomstate.Manager
}

func newSession(t *testing.T) *testSession {
	var ts testSession

	// fake dbs
	ts.AliasesDB = new(mockdb.FakeAliasesService)
	ts.DeniedKeysDB = new(mockdb.FakeDeniedKeysService)
	ts.MembersDB = new(mockdb.FakeMembersService)
	ts.PinnedDB = new(mockdb.FakePinnedNoticesService)
	ts.NoticeDB = new(mockdb.FakeNoticesService)
	ts.InvitesDB = new(mockdb.FakeInvitesService)

	log, _ := logtest.KitLogger("admin", t)
	ctx := context.TODO()
	ts.RoomState = roomstate.NewManager(ctx, log)

	ts.Domain = randutil.String(10)

	ts.Router = router.CompleteApp()

	// instantiate the urlTo helper (constructs urls for us!)
	// the cookiejar in our custom http/tester needs a non-empty domain and scheme
	urlTo := web.NewURLTo(ts.Router)
	ts.URLTo = func(name string, vals ...interface{}) *url.URL {
		testURL := urlTo(name, vals...)
		if testURL == nil {
			t.Fatalf("no URL for %s", name)
		}
		testURL.Host = ts.Domain
		testURL.Scheme = "https" // fake
		return testURL
	}

	// fake user
	ts.User = roomdb.Member{
		ID:   1234,
		Role: roomdb.RoleModerator,
	}

	testPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testPath)

	i18ntesting.WriteReplacement(t)

	testRepo := repo.New(testPath)
	locHelper, err := i18n.New(testRepo)
	if err != nil {
		t.Fatal(err)
	}

	authKey := make([]byte, 64)
	rand.Read(authKey)
	encKey := make([]byte, 32)
	rand.Read(encKey)

	sessionsPath := filepath.Join(testPath, "sessions")
	os.MkdirAll(sessionsPath, 0700)
	fsStore := sessions.NewFilesystemStore(sessionsPath, authKey, encKey)

	flashHelper := weberrs.NewFlashHelper(fsStore, locHelper)

	// setup rendering

	// TODO: make testing utils and move these there
	testFuncs := web.TemplateFuncs(ts.Router)
	testFuncs["current_page_is"] = func(routeName string) bool { return true }
	testFuncs["is_logged_in"] = func() *roomdb.Member { return &ts.User }
	testFuncs["urlToNotice"] = func(name string) string { return "" }
	testFuncs["relative_time"] = func(when time.Time) string { return humanize.Time(when) }

	renderOpts := []render.Option{
		render.SetLogger(log),
		render.BaseTemplates("base.tmpl", "menu.tmpl", "flashes.tmpl"),
		render.AddTemplates(append(HTMLTemplates, "error.tmpl")...),
		render.ErrorTemplate("error.tmpl"),
		render.FuncMap(testFuncs),
	}
	renderOpts = append(renderOpts, locHelper.GetRenderFuncs()...)

	r, err := render.New(web.Templates, renderOpts...)
	if err != nil {
		t.Fatal(errors.Wrap(err, "setup: render init failed"))
	}

	handler := Handler(
		ts.Domain,
		r,
		ts.RoomState,
		flashHelper,
		Databases{
			Aliases:       ts.AliasesDB,
			DeniedKeys:    ts.DeniedKeysDB,
			Members:       ts.MembersDB,
			Invites:       ts.InvitesDB,
			Notices:       ts.NoticeDB,
			PinnedNotices: ts.PinnedDB,
		},
	)

	handler = members.MiddlewareForTests(ts.User)(handler)

	ts.Mux = http.NewServeMux()
	ts.Mux.Handle("/", handler)

	ts.Client = tester.New(ts.Mux, t)

	return &ts
}
