// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package netwraputil

import (
	"net"
	"testing"

	"github.com/ssbc/go-ssb-room/v2/internal/maybemod/keys"
	"github.com/ssbc/go-ssb-room/v2/internal/network"
	"github.com/stretchr/testify/require"
)

func TestSpoof(t *testing.T) {
	r := require.New(t)

	rc, wc := net.Pipe()

	kp, err := keys.NewKeyPair(nil)
	r.NoError(err)

	wrap := SpoofRemoteAddress(kp.Feed.PubKey())

	wrapped, err := wrap(wc)
	r.NoError(err)

	ref, err := network.GetFeedRefFromAddr(wrapped.RemoteAddr())
	r.NoError(err)
	r.True(ref.Equal(kp.Feed))

	wc.Close()
	rc.Close()
}
