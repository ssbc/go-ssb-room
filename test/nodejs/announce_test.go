// SPDX-License-Identifier: MIT

package nodejs_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"go.cryptoscope.co/netwrap"
	"go.cryptoscope.co/secretstream"
)

// all js end-to-end test as a sanity check
func TestAllJSEndToEnd(t *testing.T) {
	// defer leakcheck.Check(t)
	r := require.New(t)

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	// alice is the server now
	alice, port := ts.startJSBotAsServer("alice", "./testscripts/server.js")

	aliceAddr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: port,
	}

	bob := ts.startJSClient("bob", "./testscripts/simple_client.js",
		aliceAddr,
		*alice,
	)

	// claire wants to connect to bob through alice

	// nasty multiserver-addr hack
	var roomHandle bytes.Buffer
	roomHandle.WriteString("tunnel:")
	roomHandle.WriteString(alice.Ref())
	roomHandle.WriteString(":")
	roomHandle.WriteString(bob.Ref())
	roomHandle.WriteString("~shs:")
	roomHandle.WriteString(base64.StdEncoding.EncodeToString(bob.ID))

	// write the handle to the testrun folder of the bot
	handleFile := filepath.Join("testrun", t.Name(), "claire", "endpoint_through_room.txt")
	os.MkdirAll(filepath.Dir(handleFile), 0700)
	err := ioutil.WriteFile(handleFile, roomHandle.Bytes(), 0700)
	r.NoError(err)

	time.Sleep(1000 * time.Millisecond)

	claire := ts.startJSClient("claire", "./testscripts/simple_client_opening_tunnel.js",
		aliceAddr,
		*alice,
	)
	t.Log("this is claire:", claire.Ref())

	time.Sleep(20 * time.Second)

	ts.wait()
}

func TestJSClient(t *testing.T) {
	// defer leakcheck.Check(t)
	r := require.New(t)

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	srv := ts.startGoServer()

	alice := ts.startJSClient("alice", "./testscripts/simple_client.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)
	srv.Allow(alice, true)

	var roomHandle bytes.Buffer
	roomHandle.WriteString("tunnel:")
	roomHandle.WriteString(srv.Whoami().Ref())
	roomHandle.WriteString(":")
	roomHandle.WriteString(alice.Ref())
	roomHandle.WriteString("~shs:")
	roomHandle.WriteString(base64.StdEncoding.EncodeToString(alice.ID))

	// write the handle to the testrun folder of the bot
	handleFile := filepath.Join("testrun", t.Name(), "bob", "endpoint_through_room.txt")
	os.MkdirAll(filepath.Dir(handleFile), 0700)
	err := ioutil.WriteFile(handleFile, roomHandle.Bytes(), 0700)
	r.NoError(err)

	time.Sleep(1500 * time.Millisecond)
	bob := ts.startJSClient("bob", "./testscripts/simple_client_opening_tunnel.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	srv.Allow(bob, true)

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

	// alice is the server now
	alice, port := ts.startJSBotAsServer("alice", "./testscripts/server.js")

	// a 2nd js instance but as a client
	aliceAddr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: port,
	}

	// now connect our go client
	client := ts.startGoServer()
	client.Allow(*alice, true)

	var roomHandle bytes.Buffer
	roomHandle.WriteString("tunnel:")
	roomHandle.WriteString(alice.Ref())
	roomHandle.WriteString(":")
	roomHandle.WriteString(client.Whoami().Ref())
	roomHandle.WriteString("~shs:")
	roomHandle.WriteString(base64.StdEncoding.EncodeToString(client.Whoami().ID))

	// write the handle to the testrun folder of the bot
	handleFile := filepath.Join("testrun", t.Name(), "bob", "endpoint_through_room.txt")
	os.MkdirAll(filepath.Dir(handleFile), 0700)
	err := ioutil.WriteFile(handleFile, roomHandle.Bytes(), 0700)
	r.NoError(err)

	bob := ts.startJSClient("bob", "./testscripts/simple_client_opening_tunnel.js",
		aliceAddr,
		*alice,
	)
	t.Log("started bob:", bob.Ref())

	client.Allow(bob, true)

	// connect to alice
	aliceShsAddr := netwrap.WrapAddr(aliceAddr, secretstream.Addr{PubKey: alice.ID})

	ctx, connCancel := context.WithCancel(context.TODO())
	err = client.Network.Connect(ctx, aliceShsAddr)
	defer connCancel()
	r.NoError(err, "connect #1 failed")

	// this might fail if the previous node process is still running...
	// TODO: properly write cleanup

	time.Sleep(2 * time.Second)

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
	a.False(ret, "would assume these are true but..?")

	select {
	case <-time.After(3 * time.Second):
		t.Error("timeout")
	case got := <-newMemberChan:
		t.Log("received join?")
		t.Log(got)
	}
	time.Sleep(5 * time.Second)

	err = srvEdp.Async(ctx, &ret, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "leave"})
	r.NoError(err)
	a.False(ret, "would assume these are true but..?")

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
