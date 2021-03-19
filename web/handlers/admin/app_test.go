// SPDX-License-Identifier: MIT

package admin

import (
	"context"
	"math/rand"
	"net/http"
	"testing"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/gorilla/mux"
	"github.com/pkg/errors"
	"go.mindeco.de/http/render"
	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/user"
)

type testSession struct {
	Mux    *http.ServeMux
	Client *tester.Tester
	Router *mux.Router

	AliasesDB *mockdb.FakeAliasesService
	MembersDB *mockdb.FakeMembersService
	PinnedDB  *mockdb.FakePinnedNoticesService
	NoticeDB  *mockdb.FakeNoticesService
	InvitesDB *mockdb.FakeInvitesService

	User *roomdb.Member

	Domain string

	RoomState *roomstate.Manager
}

func newSession(t *testing.T) *testSession {
	var ts testSession

	// fake dbs
	ts.AliasesDB = new(mockdb.FakeAliasesService)
	ts.MembersDB = new(mockdb.FakeMembersService)
	ts.PinnedDB = new(mockdb.FakePinnedNoticesService)
	ts.NoticeDB = new(mockdb.FakeNoticesService)
	ts.InvitesDB = new(mockdb.FakeInvitesService)

	log, _ := logtest.KitLogger("admin", t)
	ctx := context.TODO()
	ts.RoomState = roomstate.NewManager(ctx, log)

	ts.Router = router.CompleteApp()

	ts.Domain = randomString(10)

	// fake user
	ts.User = &roomdb.Member{
		ID:       1234,
		Nickname: "room mate",
		Role:     roomdb.RoleModerator,
	}

	// setup rendering

	// TODO: make testing utils and move these there
	testFuncs := web.TemplateFuncs(ts.Router)
	testFuncs["i18n"] = func(msgID string) string { return msgID }
	testFuncs["i18npl"] = func(msgID string, count int, _ ...interface{}) string {
		if count == 1 {
			return msgID + "Singular"
		}
		return msgID + "Plural"
	}
	testFuncs["current_page_is"] = func(routeName string) bool { return true }
	testFuncs["is_logged_in"] = func() *roomdb.Member { return ts.User }
	testFuncs["urlToNotice"] = func(name string) string { return "" }
	testFuncs["relative_time"] = func(when time.Time) string { return humanize.Time(when) }

	r, err := render.New(web.Templates,
		render.SetLogger(log),
		render.BaseTemplates("base.tmpl", "menu.tmpl"),
		render.AddTemplates(append(HTMLTemplates, "error.tmpl")...),
		render.ErrorTemplate("error.tmpl"),
		render.FuncMap(testFuncs),
	)
	if err != nil {
		t.Fatal(errors.Wrap(err, "setup: render init failed"))
	}

	ts.Mux = http.NewServeMux()

	handler := Handler(
		ts.Domain,
		r,
		ts.RoomState,
		Databases{
			Aliases:       ts.AliasesDB,
			Members:       ts.MembersDB,
			Invites:       ts.InvitesDB,
			Notices:       ts.NoticeDB,
			PinnedNotices: ts.PinnedDB,
		},
	)

	handler = user.MiddlewareForTests(ts.User)(handler)

	ts.Mux.Handle("/", handler)

	ts.Client = tester.New(ts.Mux, t)

	return &ts
}

// utils

func randomString(n int) string {
	var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

	s := make([]rune, n)
	for i := range s {
		s[i] = letters[rand.Intn(len(letters))]
	}
	return string(s)
}
