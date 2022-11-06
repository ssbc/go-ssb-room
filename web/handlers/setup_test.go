// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"bytes"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"go.mindeco.de/http/tester"
	"go.mindeco.de/logging/logtest"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/ssbc/go-ssb-room/v2/internal/network/mocked"
	"github.com/ssbc/go-ssb-room/v2/internal/repo"
	"github.com/ssbc/go-ssb-room/v2/internal/signinwithssb"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/roomdb/mockdb"
	"github.com/ssbc/go-ssb-room/v2/roomstate"
	"github.com/ssbc/go-ssb-room/v2/web"
	"github.com/ssbc/go-ssb-room/v2/web/i18n/i18ntesting"
	"github.com/ssbc/go-ssb-room/v2/web/router"
)

type testSession struct {
	Mux    *http.ServeMux
	Client *tester.Tester
	URLTo  web.URLMaker

	netInfo network.ServerEndpointDetails

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

	roomID, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("test"), 8), refs.RefAlgoFeedSSB1)
	if err != nil {
		t.Error(err)
	}

	ts.NetworkInfo = network.ServerEndpointDetails{
		Domain:              "localhost",
		PortHTTPS:           443,
		ListenAddressMUXRPC: ":8008",
		RoomID:              roomID,
	}

	log, _ := logtest.KitLogger("complete", t)

	ts.RoomState = roomstate.NewManager(log)

	// instantiate the urlTo helper (constructs urls for us!)
	// the cookiejar in our custom http/tester needs a non-empty domain and scheme
	mkUrl := web.NewURLTo(router.CompleteApp(), ts.NetworkInfo)
	ts.URLTo = func(name string, vals ...interface{}) *url.URL {
		u := mkUrl(name, vals...)
		if u.Path == "" || u.Host == "" {
			t.Fatal("failed to make URL for: ", name, vals)
		}
		return u
	}

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
