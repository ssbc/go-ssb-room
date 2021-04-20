// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"testing"
	"time"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	refs "go.mindeco.de/ssb-refs"
)

func TestTunnelServerSimple(t *testing.T) {
	// defer leakcheck.Check(t)
	ctx, cancel := context.WithCancel(context.Background())
	theBots := createServerAndBots(t, ctx, 2)

	r := require.New(t)
	a := assert.New(t)

	const (
		indexSrv = iota
		indexA
		indexB
	)

	serv := theBots[indexSrv].srv
	botA := theBots[indexA].srv
	botB := theBots[indexB].srv

	// only allow A
	theBots[indexSrv].srv.Members.Add(ctx, botA.Whoami(), roomdb.RoleMember)

	// allow bots to dial the remote
	theBots[indexA].srv.Members.Add(ctx, serv.Whoami(), roomdb.RoleMember)
	theBots[indexB].srv.Members.Add(ctx, serv.Whoami(), roomdb.RoleMember)

	// dial up B->A and C->A

	// should work (we allowed A)
	err := botA.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect A to the Server")

	// shouldn't work (we did not allowed A)
	err = botB.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect B to the Server") // we dont see an error because it just establishes the tcp connection

	t.Log("letting handshaking settle..")
	time.Sleep(1 * time.Second)

	var srvWho struct {
		ID refs.FeedRef
	}

	endpointB, has := botB.Network.GetEndpointFor(serv.Whoami())
	r.False(has, "botB has an endpoint for the server")
	if endpointB != nil {
		a.Nil(endpointB, "should not have an endpoint on B")
		err = endpointB.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
		r.Error(err)
		t.Log(srvWho.ID.Ref())
	}

	endpointA, has := botA.Network.GetEndpointFor(serv.Whoami())
	r.True(has, "botA has no endpoint for the server")

	err = endpointA.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
	r.NoError(err)

	t.Log("server whoami:", srvWho.ID.Ref())
	a.True(serv.Whoami().Equal(&srvWho.ID))

	// start testing basic room stuff
	var yes bool
	err = endpointA.Async(ctx, &yes, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "isRoom"})
	r.NoError(err)
	a.True(yes, "srv is not a room?!")

	var ts int
	err = endpointA.Async(ctx, &ts, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "ping"})
	r.NoError(err)
	t.Log("ping:", ts)

	// cleanup
	cancel()

}

func TestRoomAnnounce(t *testing.T) {
	// defer leakcheck.Check(t)
	ctx, cancel := context.WithCancel(context.Background())
	theBots := createServerAndBots(t, ctx, 2)

	r := require.New(t)
	a := assert.New(t)

	const (
		indexSrv = iota
		indexA
		indexB
	)

	serv := theBots[indexSrv].srv
	botA := theBots[indexA].srv
	botB := theBots[indexB].srv

	// allow both clients
	theBots[indexSrv].srv.Members.Add(ctx, botA.Whoami(), roomdb.RoleMember)
	theBots[indexSrv].srv.Members.Add(ctx, botB.Whoami(), roomdb.RoleMember)

	// allow bots to dial the remote
	theBots[indexA].srv.Members.Add(ctx, serv.Whoami(), roomdb.RoleMember)
	theBots[indexB].srv.Members.Add(ctx, serv.Whoami(), roomdb.RoleMember)

	// should work (we allowed A)
	err := botA.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect A to the Server")

	// shouldn't work (we did not allowed A)
	err = botB.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect B to the Server") // we dont see an error because it just establishes the tcp connection

	t.Log("letting handshaking settle..")
	time.Sleep(1 * time.Second)

	var srvWho struct {
		ID refs.FeedRef
	}
	endpointA, has := botA.Network.GetEndpointFor(serv.Whoami())
	r.True(has, "botA has no endpoint for the server")

	endpointB, has := botB.Network.GetEndpointFor(serv.Whoami())
	r.True(has, "botB has no endpoint for the server!")

	err = endpointA.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
	r.NoError(err)
	a.True(serv.Whoami().Equal(&srvWho.ID))

	err = endpointB.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
	r.NoError(err)
	a.True(serv.Whoami().Equal(&srvWho.ID))

	// let B listen for changes
	newRoomMember, err := endpointB.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
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
	err = endpointA.Async(ctx, &ret, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "announce"})
	r.NoError(err)
	a.False(ret) // <ascii-shrugg>

	select {
	case <-time.After(10 * time.Second):
		t.Error("timeout")
	case got := <-newMemberChan:
		t.Log("received join?")
		t.Log(got)
	}
	time.Sleep(5 * time.Second)

	err = endpointA.Async(ctx, &ret, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "leave"})
	r.NoError(err)
	a.False(ret) // <ascii-shrugg>

	select {
	case <-time.After(10 * time.Second):
		t.Error("timeout")
	case got := <-newMemberChan:
		t.Log("received leave?")
		t.Log(got)
	}

	// cleanup
	cancel()
}
