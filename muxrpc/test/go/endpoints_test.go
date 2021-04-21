package go_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/debug"
	"go.cryptoscope.co/netwrap"
	"go.cryptoscope.co/secretstream"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
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
	go logStream(ts, carlEndpointsSerc, "carl", announcementsForCarl)
	time.Sleep(1 * time.Second) // give some time to process new events

	_, seen := announcementsForCarl[carlFeed.Ref()]
	a.True(seen, "carl saw himself")

	// let alf join the room
	alfEndpointsSerc, err := alf.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	announcementsForAlf := make(announcements)
	go logStream(ts, alfEndpointsSerc, "alf", announcementsForAlf)
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
	go logStream(ts, breEndpointsSrc, "bre", announcementsForBre)

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

func logStream(ts *testSession, src *muxrpc.ByteSource, who string, a announcements) {
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

func (ts *testSession) makeTestClient(name string) (muxrpc.Endpoint, refs.FeedRef) {
	r := require.New(ts.t)

	// create a fresh keypairs for the clients
	client, err := keys.NewKeyPair(nil)
	r.NoError(err)

	ts.t.Log(name, "is", client.Feed.ShortRef())

	// add it as a memeber
	memberID, err := ts.srv.Members.Add(ts.ctx, client.Feed, roomdb.RoleMember)
	r.NoError(err)
	ts.t.Log(name, "is member ID:", memberID)

	// default app key for the secret-handshake connection
	ak, err := base64.StdEncoding.DecodeString("1KHLiKZvAvjbY1ziZEHMXawbCEIM6qwjCDm3VYRan/s=")
	r.NoError(err)

	// create a shs client to authenticate and encrypt the connection
	clientSHS, err := secretstream.NewClient(client.Pair, ak)
	r.NoError(err)

	// returns a new connection that went through shs and does boxstream

	tcpAddr := netwrap.GetAddr(ts.srv.Network.GetListenAddr(), "tcp")

	authedConn, err := netwrap.Dial(tcpAddr, clientSHS.ConnWrapper(ts.srv.Whoami().PubKey()))
	r.NoError(err)

	var muxMock muxrpc.FakeHandler

	testPath := filepath.Join("testrun", ts.t.Name())
	debugConn := debug.Dump(filepath.Join(testPath, "client-"+name), authedConn)
	pkr := muxrpc.NewPacker(debugConn)

	wsEndpoint := muxrpc.Handle(pkr, &muxMock, muxrpc.WithContext(ts.ctx))

	srv := wsEndpoint.(muxrpc.Server)
	ts.serveGroup.Go(func() error {
		err = srv.Serve()
		if err != nil {
			ts.t.Logf("mux server %s error: %v", name, err)
		}
		return err
	})

	// check we are talking to a room
	var yup bool
	err = wsEndpoint.Async(ts.ctx, &yup, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "isRoom"})
	r.NoError(err)
	r.True(yup, "server is not a room?")

	return wsEndpoint, client.Feed
}
