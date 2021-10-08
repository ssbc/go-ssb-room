// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"bytes"
	"context"
	"crypto/rand"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/sessions"
	"github.com/pkg/errors"
	"go.mindeco.de/http/render"
	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/randutil"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomstate"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	weberrs "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/i18n/i18ntesting"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
)

type testSession struct {
	netInfo network.ServerEndpointDetails
	Mux     *http.ServeMux

	Client *tester.Tester

	URLTo web.URLMaker

	AliasesDB    *mockdb.FakeAliasesService
	ConfigDB     *mockdb.FakeRoomConfig
	DeniedKeysDB *mockdb.FakeDeniedKeysService
	FallbackDB   *mockdb.FakeAuthFallbackService
	InvitesDB    *mockdb.FakeInvitesService
	NoticeDB     *mockdb.FakeNoticesService
	MembersDB    *mockdb.FakeMembersService
	PinnedDB     *mockdb.FakePinnedNoticesService

	User roomdb.Member

	RoomState *roomstate.Manager
}

var pubKeyCount byte

func generatePubKey() refs.FeedRef {
	pk := refs.FeedRef{Algo: "ed25519", ID: bytes.Repeat([]byte{pubKeyCount}, 32)}
	pubKeyCount++
	return pk
}

func newSession(t *testing.T) *testSession {
	var ts testSession

	// fake dbs
	ts.AliasesDB = new(mockdb.FakeAliasesService)
	ts.ConfigDB = new(mockdb.FakeRoomConfig)
	// default mode for all tests
	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeCommunity, nil)
	ts.ConfigDB.GetDefaultLanguageReturns("en", nil)
	ts.DeniedKeysDB = new(mockdb.FakeDeniedKeysService)
	ts.FallbackDB = new(mockdb.FakeAuthFallbackService)
	ts.MembersDB = new(mockdb.FakeMembersService)
	ts.PinnedDB = new(mockdb.FakePinnedNoticesService)
	ts.NoticeDB = new(mockdb.FakeNoticesService)
	ts.InvitesDB = new(mockdb.FakeInvitesService)

	log, _ := logtest.KitLogger("admin", t)
	ts.RoomState = roomstate.NewManager(log)

	ts.netInfo = network.ServerEndpointDetails{
		Domain: randutil.String(10),
		RoomID: generatePubKey(),

		UseSubdomainForAliases: true,
	}

	// instantiate the urlTo helper (constructs urls for us!)
	// the cookiejar in our custom http/tester needs a non-empty domain and scheme
	router := router.CompleteApp()
	urlTo := web.NewURLTo(router, ts.netInfo)
	ts.URLTo = func(name string, vals ...interface{}) *url.URL {
		testURL := urlTo(name, vals...)
		if testURL == nil {
			t.Fatalf("no URL for %s", name)
		}
		testURL.Host = ts.netInfo.Domain
		testURL.Scheme = "https" // fake
		return testURL
	}

	// fake user
	ts.User = roomdb.Member{
		ID:     1234,
		Role:   roomdb.RoleModerator,
		PubKey: generatePubKey(),
	}

	testPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testPath)

	i18ntesting.WriteReplacement(t)

	testRepo := repo.New(testPath)
	locHelper, err := i18n.New(testRepo, ts.ConfigDB)
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

	// setup rendering
	flashHelper := weberrs.NewFlashHelper(fsStore, locHelper)

	// template funcs
	// TODO: make testing utils and move these there
	testFuncs := web.TemplateFuncs(router, ts.netInfo)
	testFuncs["current_page_is"] = func(routeName string) bool { return true }
	testFuncs["is_logged_in"] = func() *roomdb.Member { return &ts.User }
	testFuncs["urlToNotice"] = func(name string) string { return "" }
	testFuncs["language_count"] = func() int { return 1 }
	testFuncs["list_languages"] = func(*url.URL, string) string { return "" }
	testFuncs["privacy_mode_is"] = func(mode string) bool {
		pm, err := ts.ConfigDB.GetPrivacyMode(context.TODO())
		if err != nil {
			t.Fatal(err)
		}

		return pm.String() == mode
	}
	testFuncs["member_is_elevated"] = func() bool { return ts.User.Role == roomdb.RoleAdmin || ts.User.Role == roomdb.RoleModerator }
	testFuncs["member_is_admin"] = func() bool { return ts.User.Role == roomdb.RoleAdmin }
	testFuncs["member_can"] = func(what string) (bool, error) {
		actionCheck, has := members.AllowedActions(what)
		if !has {
			return false, fmt.Errorf("unrecognized action: %s", what)
		}

		pm, err := ts.ConfigDB.GetPrivacyMode(context.TODO())
		if err != nil {
			return false, err
		}

		return actionCheck(pm, ts.User.Role), nil
	}
	testFuncs["list_languages"] = func(*url.URL, string) string { return "" }
	testFuncs["relative_time"] = func(when time.Time) string { return humanize.Time(when) }

	eh := weberrs.NewErrorHandler(locHelper, flashHelper)

	renderOpts := []render.Option{
		render.SetLogger(log),
		render.BaseTemplates("base.tmpl", "menu.tmpl", "flashes.tmpl"),
		render.AddTemplates(append(HTMLTemplates, "error.tmpl")...),
		render.SetErrorHandler(eh.Handle),
		render.FuncMap(testFuncs),
	}
	renderOpts = append(renderOpts, locHelper.GetRenderFuncs()...)

	r, err := render.New(web.Templates, renderOpts...)
	if err != nil {
		t.Fatal(errors.Wrap(err, "setup: render init failed"))
	}

	eh.SetRenderer(r)

	handler := Handler(
		ts.netInfo,
		r,
		ts.RoomState,
		flashHelper,
		locHelper,
		Databases{
			Aliases:       ts.AliasesDB,
			AuthFallback:  ts.FallbackDB,
			Config:        ts.ConfigDB,
			DeniedKeys:    ts.DeniedKeysDB,
			Members:       ts.MembersDB,
			Invites:       ts.InvitesDB,
			Notices:       ts.NoticeDB,
			PinnedNotices: ts.PinnedDB,
		},
	)

	handler = members.MiddlewareForTests(&ts.User)(handler)

	ts.Mux = http.NewServeMux()
	ts.Mux.Handle("/", handler)

	ts.Client = tester.New(ts.Mux, t)

	return &ts
}
