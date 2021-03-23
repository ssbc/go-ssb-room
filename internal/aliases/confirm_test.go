// SPDX-License-Identifier: MIT

package aliases

import (
	"bytes"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/stretchr/testify/require"
	refs "go.mindeco.de/ssb-refs"
)

func TestConfirmation(t *testing.T) {
	r := require.New(t)

	// this is our room, it's not a valid feed but thats fine for this test
	roomID := refs.FeedRef{
		ID:   bytes.Repeat([]byte("test"), 8),
		Algo: "test",
	}

	// to make the test deterministic, decided by fair dice roll.
	seed := bytes.Repeat([]byte("yeah"), 8)
	// our user, who will sign the registration
	userKeyPair, err := keys.NewKeyPair(bytes.NewReader(seed))
	r.NoError(err)

	// create and fill out the registration for an alias (in this case the name of the test)
	var valid Registration
	valid.RoomID = roomID
	valid.UserID = userKeyPair.Feed
	valid.Alias = t.Name()

	// internal function to create the registration string
	msg := valid.createRegistrationMessage()
	want := "=room-alias-registration:@dGVzdHRlc3R0ZXN0dGVzdHRlc3R0ZXN0dGVzdHRlc3Q=.test:@Rt2aJrtOqWXhBZ5/vlfzeWQ9Bj/z6iT8CMhlr2WWlG4=.ed25519:TestConfirmation"
	r.Equal(want, string(msg))

	// create the signed confirmation
	confirmation := valid.Sign(userKeyPair.Pair.Secret)

	yes := confirmation.Verify()
	r.True(yes, "should be valid for this room and feed")

	// make up another id for the invalid test(s)
	otherID := refs.FeedRef{
		ID:   bytes.Repeat([]byte("nope"), 8),
		Algo: "test",
	}

	confirmation.RoomID = otherID
	yes = confirmation.Verify()
	r.False(yes, "should not be valid for another room")

	confirmation.RoomID = roomID // restore
	confirmation.UserID = otherID
	yes = confirmation.Verify()
	r.False(yes, "should not be valid for this room but another feed")

	// puncture the signature to emulate an invalid one
	confirmation.Signature[0] = confirmation.Signature[0] ^ 1

	yes = confirmation.Verify()
	r.False(yes, "should not be valid anymore")

}
