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
	testCases := []struct {
		Name               string
		PrivacyMode        roomdb.PrivacyMode
		ExternalCanConnect bool
		ExternalCanList    bool
	}{
		{
			Name:               "open",
			PrivacyMode:        roomdb.ModeOpen,
			ExternalCanConnect: true,
			ExternalCanList:    true,
		},
		{
			Name:               "community",
			PrivacyMode:        roomdb.ModeCommunity,
			ExternalCanConnect: true,
			ExternalCanList:    false,
		},
		{
			Name:               "restricted",
			PrivacyMode:        roomdb.ModeRestricted,
			ExternalCanConnect: false,
			ExternalCanList:    false,
		},
	}

	for i := range testCases {
		testCase := testCases[i]

		t.Run(testCase.Name, func(t *testing.T) {
			t.Parallel()
			r := require.New(t)

			testInit(t)

			ctx, cancel := context.WithCancel(context.Background())
			t.Cleanup(cancel)

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

			_, err = session.srv.Members.Add(ctx, aliceKey.Feed, roomdb.RoleMember)
			r.NoError(err)

			_, err = session.srv.Members.Add(ctx, bobKey.Feed, roomdb.RoleMember)
			r.NoError(err)

			err = session.srv.Config.SetPrivacyMode(ctx, testCase.PrivacyMode)
			r.NoError(err)

			t.Run("member", func(t *testing.T) {
				t.Parallel()
				r := require.New(t)

				bobSession := makeNamedTestBot(t, "bob", ctx, append(netOpts,
					roomsrv.WithKeyPair(bobKey),
				))

				// allow bots to dial the remote
				// side-effect of re-using a room-server as the client
				_, err = bobSession.srv.Members.Add(ctx, session.srv.Whoami(), roomdb.RoleMember)
				r.NoError(err)

				err = bobSession.srv.Network.Connect(ctx, session.srv.Network.GetListenAddr())
				r.NoError(err, "connect bob to the Server")

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

				r.NoError(src.Err())
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
			})

			t.Run("external", func(t *testing.T) {
				t.Parallel()
				r := require.New(t)

				carolKey, err := keys.NewKeyPair(nil)
				r.NoError(err)

				carolSession := makeNamedTestBot(t, "carol", ctx, append(netOpts,
					roomsrv.WithKeyPair(carolKey),
				))

				// allow bots to dial the remote
				// side-effect of re-using a room-server as the client
				_, err = carolSession.srv.Members.Add(ctx, session.srv.Whoami(), roomdb.RoleMember)
				r.NoError(err)

				err = carolSession.srv.Network.Connect(ctx, session.srv.Network.GetListenAddr())
				r.NoError(err, "connect carol to the Server")

				t.Log("letting handshaking settle..")
				time.Sleep(1 * time.Second)

				clientForServer, ok := carolSession.srv.Network.GetEndpointFor(session.srv.Whoami())
				if testCase.ExternalCanConnect {
					r.True(ok)
				} else {
					r.False(ok)
					return
				}

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

				if testCase.ExternalCanList {
					r.NoError(src.Err())
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
				} else {
					r.EqualError(src.Err(), "muxrpc CallError: Error - external user are not allowed to list members: roomdb: object not found")
				}
			})
		})
	}
}
