package go_test

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/muxrpc/v2/debug"
	"go.cryptoscope.co/netwrap"
	"go.cryptoscope.co/secretstream"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
)

// we will let three clients (alf, bre, crl) join and see that the endpoint output is as expected
func TestEndpointClients(t *testing.T) {
	r := require.New(t)

	testPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testPath)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	// create the roomsrv
	ts := makeNamedTestBot(t, "server", ctx, nil)
	ctx = ts.ctx

	alf := ts.makeTestClient("alf")
	bre := ts.makeTestClient("bre")
	// carl wont announce to emulate manyverse's behavior (where tunnel.endpoints should be taken as tunnel.announce)
	carl := ts.makeTestClient("carl")

	// let alf join the room

	alfEndpointsSerc, err := alf.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	go logStream(ts, alfEndpointsSerc, "alf")

	var ok bool
	err = alf.Async(ctx, &ok, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "announce"})
	r.NoError(err)
	r.True(ok)

	time.Sleep(1 * time.Second)

	// let bre join the room
	ok = false
	err = bre.Async(ctx, &ok, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "announce"})
	r.NoError(err)
	r.True(ok)

	breEndpointsSrc, err := bre.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	go logStream(ts, breEndpointsSrc, "bre")
	time.Sleep(1 * time.Second)

	// let carl join the room
	// carl wont announce to emulate manyverse
	carlEndpointsSerc, err := carl.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	go logStream(ts, carlEndpointsSerc, "carl")

	time.Sleep(10 * time.Second)

	// terminate the clients
	alf.Terminate()
	bre.Terminate()
	carl.Terminate()

	// wait for all muxrpc serve()s to exit
	r.NoError(ts.serveGroup.Wait())
}

func logStream(ts *testSession, src *muxrpc.ByteSource, who string) {
	var refs []refs.FeedRef

	for src.Next(ts.ctx) {
		body, err := src.Bytes()
		if err != nil {
			panic(err)
		}
		// ts.t.Log(who, "got body:", string(body))

		err = json.Unmarshal(body, &refs)
		if err != nil {
			panic(err)
		}
		ts.t.Log(who, "got endpoints:", len(refs))
	}

	if err := src.Err(); err != nil {
		ts.t.Log("source errored: ", err)
		return
	}

	ts.t.Log(who, "stream closed")
}

func (ts *testSession) makeTestClient(name string) muxrpc.Endpoint {
	r := require.New(ts.t)

	// create a fresh keypairs for the clients
	client, err := keys.NewKeyPair(nil)
	r.NoError(err)

	ts.t.Log(name, "is", client.Feed.ShortRef())

	// add it as a memeber
	memberID, err := ts.srv.Members.Add(ts.ctx, client.Feed, roomdb.RoleMember)
	r.NoError(err)
	ts.t.Log("client member:", memberID)

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
		ts.t.Logf("mux server %s error: %s", name, err)
		return err
	})

	// check we are talking to a room
	var yup bool
	err = wsEndpoint.Async(ts.ctx, &yup, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "isRoom"})
	r.NoError(err)
	r.True(yup, "server is not a room?")

	return wsEndpoint
}
