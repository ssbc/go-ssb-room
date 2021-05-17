// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/tunnel/server"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	refs "go.mindeco.de/ssb-refs"
)

// this tests the new room.attendants call
// basically the same test as endpoints_test
func TestRoomAttendants(t *testing.T) {
	testInit(t)
	r := require.New(t)
	a := assert.New(t)

	testPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testPath)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	// create the roomsrv
	ts := makeNamedTestBot(t, "server", ctx, nil)
	ctx = ts.ctx

	// three new clients, connected to server automaticaly
	alf := ts.makeTestClient("alf")
	bre := ts.makeTestClient("bre")
	carl := ts.makeTestClient("carl")

	// start with carl
	// ===============
	var ok bool
	err := carl.Async(ctx, &ok, muxrpc.TypeJSON, muxrpc.Method{"room", "announce"})
	r.NoError(err)
	a.True(ok, "announce should be fine")

	carlsSource, err := carl.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"room", "attendants"})
	r.NoError(err)
	t.Log("sarah opened endpoints")

	// first message should be initial state
	a.True(carlsSource.Next(ctx))
	var initState server.AttendantsInitialState
	decodeJSONsrc(t, carlsSource, &initState)
	a.Equal("state", initState.Type)
	a.Len(initState.IDs, 1)
	a.True(initState.IDs[0].Equal(&carl.feed))

	announcementsForCarl := make(announcements)
	go logAttendantsStream(ts, carlsSource, "carl", announcementsForCarl)
	time.Sleep(1 * time.Second) // give some time to process new events

	a.Len(announcementsForCarl, 0, "none yet")

	// let alf join the room
	// =====================
	err = alf.Async(ctx, &ok, muxrpc.TypeJSON, muxrpc.Method{"room", "announce"})
	r.NoError(err)
	a.True(ok, "announce should be fine")

	alfsSource, err := alf.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"room", "attendants"})
	r.NoError(err)

	// first message should be initial state
	a.True(alfsSource.Next(ctx))
	decodeJSONsrc(t, alfsSource, &initState)
	a.Equal("state", initState.Type)
	a.Len(initState.IDs, 2)
	assertListContains(t, initState.IDs, carl.feed)
	assertListContains(t, initState.IDs, alf.feed)

	announcementsForAlf := make(announcements)
	go logAttendantsStream(ts, alfsSource, "alf", announcementsForAlf)
	time.Sleep(1 * time.Second) // give some time to process new events

	// assert what alf saw
	var seen bool
	a.Len(announcementsForAlf, 0, "none yet")

	// assert what carl saw
	_, seen = announcementsForCarl[alf.feed.Ref()]
	a.True(seen, "carl saw alf")

	// let bre join the room
	err = bre.Async(ctx, &ok, muxrpc.TypeJSON, muxrpc.Method{"room", "announce"})
	r.NoError(err)
	a.True(ok, "announce should be fine")

	bresSource, err := bre.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"room", "attendants"})
	r.NoError(err)

	// first message should be initial state
	a.True(bresSource.Next(ctx))
	decodeJSONsrc(t, bresSource, &initState)
	a.Equal("state", initState.Type)
	a.Len(initState.IDs, 3)
	assertListContains(t, initState.IDs, alf.feed)
	assertListContains(t, initState.IDs, bre.feed)
	assertListContains(t, initState.IDs, carl.feed)

	announcementsForBre := make(announcements)
	go logAttendantsStream(ts, bresSource, "bre", announcementsForBre)

	time.Sleep(1 * time.Second) // give some time to process new events

	a.Len(announcementsForBre, 0, "none yet")

	//  the two present people saw her
	_, seen = announcementsForAlf[bre.feed.Ref()]
	a.True(seen, "alf saw bre")

	_, seen = announcementsForCarl[bre.feed.Ref()]
	a.True(seen, "carl saw alf")

	// shutdown alf first
	alf.Terminate()

	time.Sleep(1 * time.Second) // give some time to process new events

	// bre and arl should have removed him

	_, seen = announcementsForBre[alf.feed.Ref()]
	a.False(seen, "alf should be gone for bre")

	_, seen = announcementsForCarl[alf.feed.Ref()]
	a.False(seen, "alf should be gone for carl")

	// terminate server and the clients
	ts.srv.Shutdown()
	bre.Terminate()
	carl.Terminate()
	ts.srv.Close()

	// wait for all muxrpc serve()s to exit
	r.NoError(ts.serveGroup.Wait())
	cancel()

}

func assertListContains(t *testing.T, lst []refs.FeedRef, who refs.FeedRef) {
	var found = false
	for _, feed := range lst {
		if feed.Equal(&who) {
			found = true
		}
	}
	if !found {
		t.Errorf("did not find %s in list of %d", who.ShortRef(), len(lst))
	}
}

func decodeJSONsrc(t *testing.T, src *muxrpc.ByteSource, val interface{}) {
	err := src.Reader(func(rd io.Reader) error {
		return json.NewDecoder(rd).Decode(val)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// consume endpoint messaes and put each peer on the passed map
func logAttendantsStream(ts *testSession, src *muxrpc.ByteSource, who string, a announcements) {
	var update server.AttendantsUpdate

	for src.Next(ts.ctx) {
		body, err := src.Bytes()
		if err != nil {
			panic(err)
		}
		// ts.t.Log(who, "got body:", string(body))

		err = json.Unmarshal(body, &update)
		if err != nil {
			panic(err)
		}
		ts.t.Log(who, "got an update:", update.Type, update.ID.ShortRef())

		switch update.Type {
		case "joined":
			a[update.ID.Ref()] = struct{}{}
		case "left":
			delete(a, update.ID.Ref())
		default:
			ts.t.Fatalf("%s: unexpected update type: %v", who, update.Type)
		}
	}

	if err := src.Err(); err != nil {
		ts.t.Log(who, "source errored: ", err)
		return
	}

	ts.t.Log(who, "stream closed")
}
