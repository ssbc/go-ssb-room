package signinwithssb

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBridge(t *testing.T) {
	a := assert.New(t)

	sb := NewSignalBridge()

	// try to use a non-existant session
	err := sb.CompleteSession("nope", false)
	a.Error(err)

	// make a new session
	updates, sc := sb.RegisterSession()

	b, err := DecodeChallengeString(sc)
	a.NoError(err)
	a.Len(b, challengeLength)

	go func() {
		err := sb.CompleteSession(sc, true)
		a.NoError(err)
	}()

	time.Sleep(time.Second / 4)

	select {
	case evt := <-updates:
		a.True(evt.Worked)
	default:
		t.Error("no updates")
	}
}
