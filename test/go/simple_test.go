package go_test

import (
	"context"
	"crypto/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"golang.org/x/sync/errgroup"

	refs "go.mindeco.de/ssb-refs"
	"go.mindeco.de/ssb-rooms/roomsrv"
)

func TestTunnelServerSimple(t *testing.T) {
	r := require.New(t)
	a := assert.New(t)

	// defer leakcheck.Check(t)
	testInit(t)

	ctx, cancel := context.WithCancel(context.Background())
	botgroup, ctx := errgroup.WithContext(ctx)

	bs := newBotServer(ctx, mainLog)

	appKey := make([]byte, 32)
	rand.Read(appKey)

	netOpts := []roomsrv.Option{
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
	}

	serv := makeNamedTestBot(t, "srv", netOpts)
	botgroup.Go(bs.Serve(serv))

	botA := makeNamedTestBot(t, "B", netOpts)
	botgroup.Go(bs.Serve(botA))

	botB := makeNamedTestBot(t, "C", netOpts)
	botgroup.Go(bs.Serve(botB))

	theBots := []*roomsrv.Server{serv, botA, botB}

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
	time.Sleep(1 * time.Second)
	for _, bot := range theBots {
		bot.Shutdown()
		r.NoError(bot.Close())
	}
	r.NoError(botgroup.Wait())
}
