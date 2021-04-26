// SPDX-License-Identifier: MIT

package nodejs_test

import (
	"bytes"
	"encoding/base64"
	"io/ioutil"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/mockdb"
	refs "go.mindeco.de/ssb-refs"
)

// legacy js end-to-end test as a sanity check
// it runs the ssb-room server against two ssb-room clients
func TestLegacyJSEndToEnd(t *testing.T) {
	// defer leakcheck.Check(t)
	r := require.New(t)

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	// alice is the server now
	alice, port := ts.startJSBotAsServer("alice", "./testscripts/legacy_server.js")

	aliceAddr := &net.TCPAddr{
		IP:   net.ParseIP("127.0.0.1"),
		Port: port,
	}

	bob := ts.startJSClient("bob", "./testscripts/legacy_client.js",
		aliceAddr,
		*alice,
	)

	// claire wants to connect to bob through alice

	// write the handle to the testrun folder of the bot
	handleFile := filepath.Join("testrun", t.Name(), "claire", "endpoint_through_room.txt")
	r.NoError(writeRoomHandleFile(*alice, bob, handleFile))

	time.Sleep(1000 * time.Millisecond)

	ts.startJSClient("claire", "./testscripts/legacy_client_opening_tunnel.js",
		aliceAddr,
		*alice,
	)

	t.Log("waiting for process exits")
	// it would be nice to have a signal here to know when the legacy client is done.
	time.Sleep(10 * time.Second)

	ts.wait()
}

// Two ssb-room clients against a Go server
func TestGoServerLegacyJSClient(t *testing.T) {
	// defer leakcheck.Check(t)
	r := require.New(t)

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	var membersDB = &mockdb.FakeMembersService{}
	var aliases = &mockdb.FakeAliasesService{}
	srv := ts.startGoServer(membersDB, aliases)
	// allow all peers (there arent any we dont want to allow)
	membersDB.GetByFeedReturns(roomdb.Member{ID: 1234}, nil)

	alice := ts.startJSClient("alice", "./testscripts/legacy_client.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	// write the handle to the testrun folder of the bot
	handleFile := filepath.Join("testrun", t.Name(), "bob", "endpoint_through_room.txt")
	r.NoError(writeRoomHandleFile(srv.Whoami(), alice, handleFile))

	time.Sleep(1500 * time.Millisecond)
	ts.startJSClient("bob", "./testscripts/legacy_client_opening_tunnel.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	t.Log("waiting for process exits")
	// it would be nice to have a signal here to know when the legacy client is done.
	time.Sleep(15 * time.Second)

	// cancels all contexts => kills all the running processes and waits
	// (everything should have exited by now!)
	ts.wait()
}

// the new ssb-room-client module (2x) against a Go room server
func TestModernJSClient(t *testing.T) {
	// defer leakcheck.Check(t)
	r := require.New(t)

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	var membersDB = &mockdb.FakeMembersService{}
	var aliasesDB = &mockdb.FakeAliasesService{}
	srv := ts.startGoServer(membersDB, aliasesDB)
	membersDB.GetByFeedReturns(roomdb.Member{ID: 1234}, nil)

	// allow all peers (there arent any we dont want to allow in this test)

	alice := ts.startJSClient("alice", "./testscripts/modern_client.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	// write the handle to the testrun folder of the bot
	handleFile := filepath.Join("testrun", t.Name(), "bob", "endpoint_through_room.txt")
	r.NoError(writeRoomHandleFile(srv.Whoami(), alice, handleFile))

	ts.startJSClient("bob", "./testscripts/modern_client_opening_tunnel.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	time.Sleep(15 * time.Second)

	ts.wait()
}

// found a nasty `throw err` in the JS stack around pull.drain. lets make sure it stays gone
// https://github.com/ssb-ngi-pointer/go-ssb-room/issues/190
func TestClientSurvivesShutdown(t *testing.T) {
	r := require.New(t)

	ts := newRandomSession(t)

	var membersDB = &mockdb.FakeMembersService{}
	var aliasesDB = &mockdb.FakeAliasesService{}
	srv := ts.startGoServer(membersDB, aliasesDB)
	membersDB.GetByFeedReturns(roomdb.Member{ID: 1234}, nil)

	alice := ts.startJSClient("alice", "./testscripts/modern_client.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)
	// write the handle to the testrun folder of the bot
	handleFile := filepath.Join("testrun", t.Name(), "bob", "endpoint_through_room.txt")
	r.NoError(writeRoomHandleFile(srv.Whoami(), alice, handleFile))

	ts.startJSClient("bob", "./testscripts/modern_client_opening_tunnel.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	// give them time to connect (which would make them pass the test)
	time.Sleep(8 * time.Second)

	// shut down server (which closes all the muxrpc streams)
	srv.Close()

	// give the node processes a moment to process what just happend (don't kill them immediatl in wait())
	// in the buggy case they will crash and exit with error code 1
	// the error is visible when running the tests with `go test -v`
	time.Sleep(2 * time.Second)

	ts.wait()
}

func writeRoomHandleFile(srv, target refs.FeedRef, filePath string) error {
	var roomHandle bytes.Buffer
	roomHandle.WriteString("tunnel:")
	roomHandle.WriteString(srv.Ref())
	roomHandle.WriteString(":")
	roomHandle.WriteString(target.Ref())
	roomHandle.WriteString("~shs:")
	roomHandle.WriteString(base64.StdEncoding.EncodeToString(target.ID))

	os.MkdirAll(filepath.Dir(filePath), 0700)
	return ioutil.WriteFile(filePath, roomHandle.Bytes(), 0700)
}
