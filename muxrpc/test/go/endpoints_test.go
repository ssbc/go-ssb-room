// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/ssbc/go-muxrpc/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	refs "github.com/ssbc/go-ssb-refs"
)

type announcements map[string]struct{}

// we will let three clients (alf, bre, crl) join and see that the endpoint output is as expected
func TestEndpointClients(t *testing.T) {
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

	// create three test clients
	alf := ts.makeTestClient("alf")
	bre := ts.makeTestClient("bre")
	carl := ts.makeTestClient("carl")

	// let carl join the room
	// carl wont announce to emulate manyverse
	carlEndpointsSrc, err := carl.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)
	t.Log("carl opened endpoints")

	announcementsForCarl := make(announcements)
	go logEndpointsStream(ts, carlEndpointsSrc, "carl", announcementsForCarl)
	time.Sleep(1 * time.Second) // give some time to process new events

	_, seen := announcementsForCarl[carl.feed.Ref()]
	a.True(seen, "carl saw himself")

	// let alf join the room
	alfEndpointsSerc, err := alf.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	announcementsForAlf := make(announcements)
	go logEndpointsStream(ts, alfEndpointsSerc, "alf", announcementsForAlf)
	time.Sleep(1 * time.Second) // give some time to process new events

	// assert what alf saw
	_, seen = announcementsForAlf[carl.feed.Ref()]
	a.True(seen, "alf saw carl")
	_, seen = announcementsForAlf[alf.feed.Ref()]
	a.True(seen, "alf saw himself")

	// assert what carl saw
	_, seen = announcementsForCarl[alf.feed.Ref()]
	a.True(seen, "carl saw alf")

	// let bre join the room
	breEndpointsSrc, err := bre.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	announcementsForBre := make(announcements)
	go logEndpointsStream(ts, breEndpointsSrc, "bre", announcementsForBre)

	time.Sleep(1 * time.Second) // give some time to process new events

	// assert bre saw the other two and herself
	_, seen = announcementsForBre[carl.feed.Ref()]
	a.True(seen, "bre saw carl")
	_, seen = announcementsForBre[alf.feed.Ref()]
	a.True(seen, "bre saw alf")
	_, seen = announcementsForBre[bre.feed.Ref()]
	a.True(seen, "bre saw herself")

	// assert the others saw bre
	_, seen = announcementsForAlf[bre.feed.Ref()]
	a.True(seen, "alf saw bre")
	_, seen = announcementsForCarl[bre.feed.Ref()]
	a.True(seen, "carl saw bre")

	// terminate server and the clients
	ts.srv.Shutdown()
	alf.Terminate()
	bre.Terminate()
	carl.Terminate()
	ts.srv.Close()

	// wait for all muxrpc serve()s to exit
	r.NoError(ts.serveGroup.Wait())
	cancel()
}

// consume endpoint messaes and put each peer on the passed map
func logEndpointsStream(ts *testSession, src *muxrpc.ByteSource, who string, a announcements) {
	var edps []refs.FeedRef

	for src.Next(ts.ctx) {
		body, err := src.Bytes()
		if err != nil {
			panic(err)
		}
		// ts.t.Log(who, "got body:", string(body))

		err = json.Unmarshal(body, &edps)
		if err != nil {
			panic(err)
		}
		ts.t.Log(who, "got endpoints:", len(edps))
		for i, f := range edps {
			ts.t.Log(who, ":", i, f.ShortRef())

			// mark as f is present
			a[f.Ref()] = struct{}{}
		}
	}

	if err := src.Err(); err != nil {
		ts.t.Log("source errored: ", err)
		return
	}

	ts.t.Log(who, "stream closed")
}
