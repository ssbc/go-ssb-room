package nodejs_test

import (
	"context"
	"net"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/netwrap"
	"go.cryptoscope.co/secretstream"
)

func TestJSClient(t *testing.T) {
	// defer leakcheck.Check(t)
	// r := require.New(t)

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	srv := ts.startGoServer()

	alice := ts.startJSBot("./testscripts/simple_client.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	srv.Allow(alice, true)

	time.Sleep(5 * time.Second)

	ts.wait()
}

func TestJSServer(t *testing.T) {
	// defer leakcheck.Check(t)
	r := require.New(t)
	a := assert.New(t)

	os.RemoveAll("testrun")

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	client := ts.startGoServer()

	// alice is the server now
	alice, port := ts.startJSBotAsServer("alice", "./testscripts/server.js")

	client.Allow(*alice, true)

	// connect to alice
	wrappedAddr := netwrap.WrapAddr(&net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: port,
	}, secretstream.Addr{PubKey: alice.ID})

	ctx, connCancel := context.WithCancel(context.TODO())
	err := client.Network.Connect(ctx, wrappedAddr)
	defer connCancel()
	r.NoError(err, "connect #1 failed")

	// this might fail if the previous node process is still running...
	// TODO: properly write cleanup

	time.Sleep(3 * time.Second)

	srvEdp, has := client.Network.GetEndpointFor(*alice)
	r.True(has, "botA has no endpoint for the server")
	t.Log("connected")

	// let B listen for changes
	newRoomMember, err := srvEdp.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
	r.NoError(err)

	newMemberChan := make(chan string)

	// read all the messages from endpoints and throw them over the channel
	go func() {
		for newRoomMember.Next(ctx) {
			body, err := newRoomMember.Bytes()
			if err != nil {
				panic(err)
			}
			newMemberChan <- string(body)
		}
		close(newMemberChan)
	}()

	// announce A
	var ret bool
	err = srvEdp.Async(ctx, &ret, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "announce"})
	r.NoError(err)
	// a.Equal("joined", ret.Action)
	a.False(ret, "would assume these are true but..?")
	// a.EqualValues(1, ret.Members, "expected just one member")

	select {
	case <-time.After(3 * time.Second):
		t.Error("timeout")
	case got := <-newMemberChan:
		t.Log("received join?")
		t.Log(got)
	}
	time.Sleep(1 * time.Second)

	err = srvEdp.Async(ctx, &ret, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "leave"})
	r.NoError(err)
	// a.Equal("left", ret.Action)
	a.False(ret, "would assume these are true but..?")
	// a.EqualValues(0, ret.Members, "expected empty rooms")

	select {
	case <-time.After(3 * time.Second):
		t.Error("timeout")
	case got := <-newMemberChan:
		t.Log("received leave?")
		t.Log(got)
	}

	srvEdp.Terminate()

	ts.wait()
}
