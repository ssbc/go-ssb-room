// SPDX-License-Identifier: MIT

package signinwithssb

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBridge(t *testing.T) {
	a := assert.New(t)

	sb := NewSignalBridge()

	// try to use a non-existant session
	err := sb.SessionFailed("nope", fmt.Errorf("just a test"))
	a.Error(err)

	// make a new session
	sc := sb.RegisterSession()

	b, err := DecodeChallengeString(sc)
	a.NoError(err)
	a.Len(b, challengeLength)

	updates, has := sb.GetEventChannel(sc)
	a.True(has)

	go func() {
		err := sb.SessionWorked(sc, "a token")
		a.NoError(err)
	}()

	time.Sleep(time.Second / 4)

	select {
	case evt := <-updates:
		a.True(evt.Worked)
		a.Equal("a token", evt.Token)
	default:
		t.Error("no updates")
	}
}
