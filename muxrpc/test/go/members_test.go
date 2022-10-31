// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"encoding/json"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/muxrpc/handlers/tunnel/server"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomsrv"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"
)

// this tests the new room.members call
func TestRoomMembers(t *testing.T) {
	testInit(t)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	r := require.New(t)

	appKey := make([]byte, 32)
	rand.Read(appKey)

	netOpts := []roomsrv.Option{
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
	}

	session := makeNamedTestBot(t, "srv", ctx, netOpts)

	aliceKey, err := keys.NewKeyPair(nil)
	r.NoError(err)

	bobKey, err := keys.NewKeyPair(nil)
	r.NoError(err)

	bobSession := makeNamedTestBot(t, "bob", ctx, append(netOpts,
		roomsrv.WithKeyPair(bobKey),
	))

	_, err = session.srv.Members.Add(ctx, aliceKey.Feed, roomdb.RoleMember)
	r.NoError(err)

	_, err = session.srv.Members.Add(ctx, bobKey.Feed, roomdb.RoleMember)
	r.NoError(err)

	// allow bots to dial the remote
	// side-effect of re-using a room-server as the client
	_, err = bobSession.srv.Members.Add(ctx, session.srv.Whoami(), roomdb.RoleMember)
	r.NoError(err)

	err = bobSession.srv.Network.Connect(ctx, session.srv.Network.GetListenAddr())
	r.NoError(err, "connect A to the Server")

	t.Log("letting handshaking settle..")
	time.Sleep(1 * time.Second)

	clientForServer, ok := bobSession.srv.Network.GetEndpointFor(session.srv.Whoami())
	r.True(ok)

	src, err := clientForServer.Source(ctx, muxrpc.TypeString, muxrpc.Method{"room", "members"})
	r.NoError(err)

	var responses []server.Member
	for src.Next(ctx) {
		bytes, err := src.Bytes()
		r.NoError(err)

		var members []server.Member
		err = json.Unmarshal(bytes, &members)
		r.NoError(err)
		responses = append(responses, members...)
	}

	r.Equal(
		[]server.Member{
			{
				ID: aliceKey.Feed,
			},
			{
				ID: bobKey.Feed,
			},
		},
		responses,
	)
}
