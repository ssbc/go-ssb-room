// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/aliases"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomsrv"
)

// technically we are usign two servers here
// but we just treat one of them as a muxrpc client
func TestAliasRegister(t *testing.T) {
	testInit(t)
	ctx, cancel := context.WithCancel(context.Background())

	r := require.New(t)
	a := assert.New(t)

	// make a random test key
	appKey := make([]byte, 32)
	rand.Read(appKey)

	netOpts := []roomsrv.Option{
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
	}

	theBots := []*testSession{}

	session := makeNamedTestBot(t, "srv", ctx, netOpts)
	theBots = append(theBots, session)

	// we need bobs key to create the signature
	bobsKey, err := keys.NewKeyPair(nil)
	r.NoError(err)

	bobSession := makeNamedTestBot(t, "bob", ctx, append(netOpts,
		roomsrv.WithKeyPair(bobsKey),
	))
	theBots = append(theBots, bobSession)

	// adds
	_, err = session.srv.Members.Add(ctx, bobSession.srv.Whoami(), roomdb.RoleMember)
	r.NoError(err)

	// allow bots to dial the remote
	// side-effect of re-using a room-server as the client
	_, err = bobSession.srv.Members.Add(ctx, session.srv.Whoami(), roomdb.RoleMember)
	r.NoError(err)

	// should work (we allowed A)
	err = bobSession.srv.Network.Connect(ctx, session.srv.Network.GetListenAddr())
	r.NoError(err, "connect A to the Server")

	t.Log("letting handshaking settle..")
	time.Sleep(1 * time.Second)

	clientForServer, ok := bobSession.srv.Network.GetEndpointFor(session.srv.Whoami())
	r.True(ok)

	t.Log("got endpoint")

	var testReg aliases.Registration
	testReg.Alias = "bob"
	testReg.RoomID = session.srv.Whoami()
	testReg.UserID = bobSession.srv.Whoami()

	confirmation := testReg.Sign(bobsKey.Pair.Secret)
	t.Logf("signature created: %x...", confirmation.Signature[:16])

	// encode the signature as base64
	sig := base64.StdEncoding.EncodeToString(confirmation.Signature) + ".sig.ed25519"

	var registerResponse string
	err = clientForServer.Async(ctx, &registerResponse, muxrpc.TypeString, muxrpc.Method{"room", "registerAlias"}, "bob", sig)
	r.NoError(err)
	a.NotEqual("", registerResponse, "response isn't empty")

	resolveURL, err := url.Parse(registerResponse)
	r.NoError(err)
	t.Log("got URL:", resolveURL)
	a.Equal("bob.srv", resolveURL.Host)
	a.Equal("", resolveURL.Path)

	// server should have the alias now
	alias, err := session.srv.Aliases.Resolve(ctx, "bob")
	r.NoError(err)

	a.Equal(confirmation.Alias, alias.Name)
	a.Equal(confirmation.Signature, alias.Signature)
	a.True(confirmation.UserID.Equal(&bobsKey.Feed))

	t.Log("alias stored")

	for _, bot := range theBots {
		bot.srv.Shutdown()
		r.NoError(bot.srv.Close())
		r.NoError(bot.serveGroup.Wait())
	}
	cancel()
}
