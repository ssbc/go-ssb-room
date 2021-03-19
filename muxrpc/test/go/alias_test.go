// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"testing"
	"time"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
	"golang.org/x/sync/errgroup"

	"github.com/ssb-ngi-pointer/go-ssb-room/aliases"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomsrv"
)

// technically we are usign two servers here
// but we just treat one of them as a muxrpc client
func TestAliasRegister(t *testing.T) {
	testInit(t)
	ctx, cancel := context.WithCancel(context.Background())
	botgroup, ctx := errgroup.WithContext(ctx)
	bs := newBotServer(ctx, mainLog)

	r := require.New(t)
	a := assert.New(t)

	// make a random test key
	appKey := make([]byte, 32)
	rand.Read(appKey)

	netOpts := []roomsrv.Option{
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
	}

	theBots := []*roomsrv.Server{}

	srvMembers, serv := makeNamedTestBot(t, "srv", netOpts)
	botgroup.Go(bs.Serve(serv))
	theBots = append(theBots, serv)

	// we need bobs key to create the signature
	bobsKey, err := keys.NewKeyPair(nil)
	r.NoError(err)

	bobsMembers, bob := makeNamedTestBot(t, "bob", append(netOpts,
		roomsrv.WithKeyPair(bobsKey),
	))
	botgroup.Go(bs.Serve(bob))
	theBots = append(theBots, bob)

	t.Cleanup(func() {
		for _, bot := range theBots {
			bot.Shutdown()
			r.NoError(bot.Close())
		}
		r.NoError(botgroup.Wait())
	})

	// adds
	_, err = srvMembers.Add(ctx, "bob", bob.Whoami(), roomdb.RoleMember)
	r.NoError(err)

	// allow bots to dial the remote
	// side-effect of re-using a room-server as the client
	_, err = bobsMembers.Add(ctx, "the-room", serv.Whoami(), roomdb.RoleMember)
	r.NoError(err)

	// should work (we allowed A)
	err = bob.Network.Connect(ctx, serv.Network.GetListenAddr())
	r.NoError(err, "connect A to the Server")

	t.Log("letting handshaking settle..")
	time.Sleep(1 * time.Second)

	clientForServer, ok := bob.Network.GetEndpointFor(serv.Whoami())
	r.True(ok)

	t.Log("got endpoint")

	var testReg aliases.Registration
	testReg.Alias = "bob"
	testReg.RoomID = serv.Whoami()
	testReg.UserID = bob.Whoami()

	confirmation := testReg.Sign(bobsKey.Pair.Secret)
	t.Logf("signature created: %x...", confirmation.Signature[:16])

	// encode the signature as base64
	sig := base64.StdEncoding.EncodeToString(confirmation.Signature) + ".sig.ed25519"

	var worked bool
	err = clientForServer.Async(ctx, &worked, muxrpc.TypeJSON, muxrpc.Method{"room", "registerAlias"}, "bob", sig)
	r.NoError(err)
	a.True(worked)

	// server should have the alias now
	alias, err := serv.Aliases.Resolve(ctx, "bob")
	r.NoError(err)

	a.Equal(confirmation.Alias, alias.Name)
	a.Equal(confirmation.Signature, alias.Signature)
	a.True(confirmation.UserID.Equal(&bobsKey.Feed))

	t.Log("alias stored")

	cancel()
}
