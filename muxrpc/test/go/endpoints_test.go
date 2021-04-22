package go_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"

	refs "go.mindeco.de/ssb-refs"
)

type announcements map[string]struct{}

// we will let three clients (alf, bre, crl) join and see that the endpoint output is as expected
func TestEndpointClients(t *testing.T) {
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
	alf, alfFeed := ts.makeTestClient("alf")
	bre, breFeed := ts.makeTestClient("bre")
	carl, carlFeed := ts.makeTestClient("carl")

	// let carl join the room
	// carl wont announce to emulate manyverse
	carlEndpointsSerc, err := carl.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)
	t.Log("carl opened endpoints")

	announcementsForCarl := make(announcements)
	go logEndpointsStream(ts, carlEndpointsSerc, "carl", announcementsForCarl)
	time.Sleep(1 * time.Second) // give some time to process new events

	_, seen := announcementsForCarl[carlFeed.Ref()]
	a.True(seen, "carl saw himself")

	// let alf join the room
	alfEndpointsSerc, err := alf.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	announcementsForAlf := make(announcements)
	go logEndpointsStream(ts, alfEndpointsSerc, "alf", announcementsForAlf)
	time.Sleep(1 * time.Second) // give some time to process new events

	// assert what alf saw
	_, seen = announcementsForAlf[carlFeed.Ref()]
	a.True(seen, "alf saw carl")
	_, seen = announcementsForAlf[alfFeed.Ref()]
	a.True(seen, "alf saw himself")

	// assert what carl saw
	_, seen = announcementsForCarl[alfFeed.Ref()]
	a.True(seen, "carl saw alf")

	// let bre join the room
	breEndpointsSrc, err := bre.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	announcementsForBre := make(announcements)
	go logEndpointsStream(ts, breEndpointsSrc, "bre", announcementsForBre)

	time.Sleep(1 * time.Second) // give some time to process new events

	// assert bre saw the other two and herself
	_, seen = announcementsForBre[carlFeed.Ref()]
	a.True(seen, "bre saw carl")
	_, seen = announcementsForBre[alfFeed.Ref()]
	a.True(seen, "bre saw alf")
	_, seen = announcementsForBre[breFeed.Ref()]
	a.True(seen, "bre saw herself")

	// assert the others saw bre
	_, seen = announcementsForAlf[breFeed.Ref()]
	a.True(seen, "alf saw bre")
	_, seen = announcementsForCarl[breFeed.Ref()]
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
