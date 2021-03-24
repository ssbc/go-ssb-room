package signinwithssb

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"
	refs "go.mindeco.de/ssb-refs"
)

func TestClientRequestString(t *testing.T) {

	server := refs.FeedRef{ID: bytes.Repeat([]byte{1}, 32), Algo: "test"}

	client := refs.FeedRef{ID: bytes.Repeat([]byte{2}, 32), Algo: "test"}

	var req ClientRequest

	req.ServerID = server
	req.ClientID = client

	req.ServerChallenge = "fooo"
	req.ClientChallenge = "barr"

	want := "=http-auth-sign-in:@AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=.test:@AgICAgICAgICAgICAgICAgICAgICAgICAgICAgICAgI=.test:fooo:barr"

	got := req.createMessage()
	assert.Equal(t, want, string(got))
}
