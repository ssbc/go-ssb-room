// SPDX-License-Identifier: MIT

package handlers

import (
	"bytes"
	"context"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network/mocked"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/mockdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/i18n/i18ntesting"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	refs "go.mindeco.de/ssb-refs"
)

type testSession struct {
	Mux    *http.ServeMux
	Client *tester.Tester
	URLTo  web.URLMaker

	// mocked dbs
	AuthDB         *mockdb.FakeAuthWithSSBService
	AuthFallbackDB *mockdb.FakeAuthFallbackService
	AuthWithSSB    *mockdb.FakeAuthWithSSBService
	AliasesDB      *mockdb.FakeAliasesService
	ConfigDB       *mockdb.FakeRoomConfig
	MembersDB      *mockdb.FakeMembersService
	InvitesDB      *mockdb.FakeInvitesService
	DeniedKeysDB   *mockdb.FakeDeniedKeysService
	PinnedDB       *mockdb.FakePinnedNoticesService
	NoticeDB       *mockdb.FakeNoticesService

	RoomState *roomstate.Manager

	MockedEndpoints *mocked.FakeEndpoints

	SignalBridge *signinwithssb.SignalBridge

	NetworkInfo network.ServerEndpointDetails
}

func setup(t *testing.T) *testSession {
	t.Parallel()
	var ts testSession

	testRepoPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepoPath)
	testRepo := repo.New(testRepoPath)

	i18ntesting.WriteReplacement(t)

	ts.AuthDB = new(mockdb.FakeAuthWithSSBService)
	ts.AuthFallbackDB = new(mockdb.FakeAuthFallbackService)
	ts.AuthWithSSB = new(mockdb.FakeAuthWithSSBService)
	ts.AliasesDB = new(mockdb.FakeAliasesService)
	ts.MembersDB = new(mockdb.FakeMembersService)
	ts.ConfigDB = new(mockdb.FakeRoomConfig)
	// default mode for all tests
	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeCommunity, nil)
	ts.InvitesDB = new(mockdb.FakeInvitesService)
	ts.DeniedKeysDB = new(mockdb.FakeDeniedKeysService)
	ts.PinnedDB = new(mockdb.FakePinnedNoticesService)
	defaultNotice := &roomdb.Notice{
		Title:   "Default Notice Title",
		Content: "Default Notice Content",
	}
	ts.PinnedDB.GetReturns(defaultNotice, nil)
	ts.NoticeDB = new(mockdb.FakeNoticesService)

	ts.MockedEndpoints = new(mocked.FakeEndpoints)

	ts.NetworkInfo = network.ServerEndpointDetails{
		Domain:     "localhost",
		PortMUXRPC: 8008,
		PortHTTPS:  443,

		RoomID: refs.FeedRef{
			ID:   bytes.Repeat([]byte("test"), 8),
			Algo: refs.RefAlgoFeedSSB1,
		},
	}

	log, _ := logtest.KitLogger("complete", t)
	ctx := context.TODO()
	ts.RoomState = roomstate.NewManager(ctx, log)

	// instantiate the urlTo helper (constructs urls for us!)
	// the cookiejar in our custom http/tester needs a non-empty domain and scheme
	ts.URLTo = web.NewURLTo(router.CompleteApp(), ts.NetworkInfo)

	ts.SignalBridge = signinwithssb.NewSignalBridge()

	h, err := New(
		log,
		testRepo,
		ts.NetworkInfo,
		ts.RoomState,
		ts.MockedEndpoints,
		ts.SignalBridge,
		Databases{
			Aliases:       ts.AliasesDB,
			AuthFallback:  ts.AuthFallbackDB,
			AuthWithSSB:   ts.AuthWithSSB,
			Config:        ts.ConfigDB,
			Members:       ts.MembersDB,
			Invites:       ts.InvitesDB,
			DeniedKeys:    ts.DeniedKeysDB,
			Notices:       ts.NoticeDB,
			PinnedNotices: ts.PinnedDB,
		},
	)
	if err != nil {
		t.Fatal("setup: handler init failed:", err)
	}

	ts.Mux = http.NewServeMux()
	ts.Mux.Handle("/", h)
	ts.Client = tester.New(ts.Mux, t)

	return &ts
}
