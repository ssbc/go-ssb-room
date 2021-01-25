package netwraputil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"go.mindeco.de/ssb-rooms/internal/maybemod/keys"
	"go.mindeco.de/ssb-rooms/internal/network"
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
