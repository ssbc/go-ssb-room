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

// This test denies connections for keys that have been added to the deny list database, DeniedKeys.
// Two peers try to connect to the server, A and B. A is a member, while B has had their key banned (added to the deny
// list).
func TestConnEstablishmentDeniedKey(t *testing.T) {
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

	serv := theBots[indexSrv].Server
	botA := theBots[indexA].Server
	botB := theBots[indexB].Server

	// allow A, deny B
	theBots[indexSrv].Members.Add(ctx, botA.Whoami(), roomdb.RoleMember)
	// since we want to verify denied keys in particular, let us both:
	// a) add B as a member
	theBots[indexSrv].Members.Add(ctx, botB.Whoami(), roomdb.RoleMember)
	// b) ban B by adding them to the DeniedKeys database
	theBots[indexSrv].Server.DeniedKeys.Add(ctx, botB.Whoami(), "rude")

	// hack: allow bots to dial the server
	theBots[indexA].Members.Add(ctx, serv.Whoami(), roomdb.RoleMember)
	theBots[indexB].Members.Add(ctx, serv.Whoami(), roomdb.RoleMember)

	// dial up B->A and C->A
	// should work (we allowed A)
	err := botA.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect A to the Server")

	// shouldn't work (we banned B)
	err = botB.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect B to the Server") // we dont see an error because it just establishes the tcp connection

	t.Log("letting handshaking settle..")
	time.Sleep(1 * time.Second)

	var srvWho struct {
		ID refs.FeedRef
	}

	endpointB, has := botB.Network.GetEndpointFor(serv.Whoami())
	r.False(has, "botB has an endpoint for the server!")
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

	cancel()
}
