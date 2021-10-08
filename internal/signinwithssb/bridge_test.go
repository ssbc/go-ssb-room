// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package signinwithssb

import (
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBridgeWorked(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	sb := NewSignalBridge()

	// try to use a non-existant session
	err := sb.SessionWorked("nope", "just a test")
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
		a.Nil(evt.Reason)
	default:
		t.Error("no updates")
	}
}

func TestBridgeFailed(t *testing.T) {
	t.Parallel()

	a := assert.New(t)

	sb := NewSignalBridge()

	// try to use a non-existant session
	testReason := fmt.Errorf("just an error")
	err := sb.SessionFailed("nope", testReason)
	a.Error(err)

	// make a new session
	sc := sb.RegisterSession()

	b, err := DecodeChallengeString(sc)
	a.NoError(err)
	a.Len(b, challengeLength)

	updates, has := sb.GetEventChannel(sc)
	a.True(has)

	go func() {
		err := sb.SessionFailed(sc, testReason)
		a.NoError(err)
	}()

	time.Sleep(time.Second / 4)

	select {
	case evt := <-updates:
		a.False(evt.Worked)
		a.Equal("", evt.Token)
		a.EqualError(testReason, evt.Reason.Error())
	default:
		t.Error("no updates")
	}
}
