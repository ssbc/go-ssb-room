// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"crypto/rand"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"golang.org/x/sync/errgroup"

	refs "go.mindeco.de/ssb-refs"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomsrv"
)

func createServerAndBots(t *testing.T, ctx context.Context, count uint) []*roomsrv.Server {
	testInit(t)
	r := require.New(t)

	botgroup, ctx := errgroup.WithContext(ctx)

	bs := newBotServer(ctx, mainLog)

	appKey := make([]byte, 32)
	rand.Read(appKey)

	netOpts := []roomsrv.Option{
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
	}
	theBots := []*roomsrv.Server{}

	serv := makeNamedTestBot(t, "srv", netOpts)
	botgroup.Go(bs.Serve(serv))
	theBots = append(theBots, serv)

	for i := uint(1); i < count+1; i++ {
		botI := makeNamedTestBot(t, fmt.Sprintf("%d", i), netOpts)
		botgroup.Go(bs.Serve(botI))
		theBots = append(theBots, botI)
	}

	t.Cleanup(func() {
		time.Sleep(1 * time.Second)
		for _, bot := range theBots {
			bot.Shutdown()
			r.NoError(bot.Close())
		}
		r.NoError(botgroup.Wait())
	})

	return theBots
}

func TestTunnelServerSimple(t *testing.T) {
	// defer leakcheck.Check(t)
	ctx, cancel := context.WithCancel(context.Background())
	theBots := createServerAndBots(t, ctx, 2)

	r := require.New(t)
	a := assert.New(t)

	serv := theBots[0]

	botA := theBots[1]
	botB := theBots[2]

	// only allow B to dial A
	serv.Allow(botA.Whoami(), true)

	// allow bots to dial the remote
	botA.Allow(serv.Whoami(), true)
	botB.Allow(serv.Whoami(), true)

	// dial up B->A and C->A

	// should work (we allowed A)
	err := botA.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect A to the Server")

	// shouldn't work (we did not allowed A)
	err = botB.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect B to the Server") // we dont see an error because it just establishes the tcp connection

	// t.Log("letting handshaking settle..")
	// time.Sleep(1 * time.Second)

	var srvWho struct {
		ID refs.FeedRef
	}

	edpOfB, has := botB.Network.GetEndpointFor(serv.Whoami())
	r.False(has, "botB has an endpoint for the server!")
	if edpOfB != nil {
		a.Nil(edpOfB, "should not have an endpoint on B")
		err = edpOfB.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
		r.Error(err)
		t.Log(srvWho.ID.Ref())
	}

	edpOfA, has := botA.Network.GetEndpointFor(serv.Whoami())
	r.True(has, "botA has no endpoint for the server")

	err = edpOfA.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
	r.NoError(err)

	t.Log("server whoami:", srvWho.ID.Ref())
	a.True(serv.Whoami().Equal(&srvWho.ID))

	// start testing basic room stuff
	var yes bool
	err = edpOfA.Async(ctx, &yes, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "isRoom"})
	r.NoError(err)
	a.True(yes, "srv is not a room?!")

	var ts int
	err = edpOfA.Async(ctx, &ts, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "ping"})
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

	serv := theBots[0]

	botA := theBots[1]
	botB := theBots[2]

	// only allow B to dial A
	serv.Allow(botA.Whoami(), true)
	serv.Allow(botB.Whoami(), true)

	// allow bots to dial the remote
	botA.Allow(serv.Whoami(), true)
	botB.Allow(serv.Whoami(), true)

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
	edpOfA, has := botA.Network.GetEndpointFor(serv.Whoami())
	r.True(has, "botA has no endpoint for the server")

	edpOfB, has := botB.Network.GetEndpointFor(serv.Whoami())
	r.True(has, "botB has no endpoint for the server!")

	err = edpOfA.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
	r.NoError(err)
	a.True(serv.Whoami().Equal(&srvWho.ID))

	err = edpOfB.Async(ctx, &srvWho, muxrpc.TypeJSON, muxrpc.Method{"whoami"})
	r.NoError(err)
	a.True(serv.Whoami().Equal(&srvWho.ID))

	// let B listen for changes
	newRoomMember, err := edpOfB.Source(ctx, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "endpoints"})
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
	err = edpOfA.Async(ctx, &ret, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "announce"})
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

	err = edpOfA.Async(ctx, &ret, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "leave"})
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
