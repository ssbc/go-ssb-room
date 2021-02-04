package netwraputil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/network"
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
	r.True(ref.Equal(&kp.Feed))

	wc.Close()
	rc.Close()
}
