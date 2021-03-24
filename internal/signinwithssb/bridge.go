// SPDX-License-Identifier: MIT

package signinwithssb

import (
	"fmt"
	"sync"
	"time"
)

// SignalBridge implements a way for muxrpc and http handlers to communicate about SIWSSB events
type SignalBridge struct {
	mu *sync.Mutex

	sessions sessionMap
}

type sessionMap map[string]chan Event

type Event struct {
	Worked bool
	Token  string
}

// NewSignalBridge returns a new SignalBridge
func NewSignalBridge() *SignalBridge {
	return &SignalBridge{
		mu:       new(sync.Mutex),
		sessions: make(sessionMap),
	}
}

// RegisterSession registers a new session on the bridge.
// It returns a fresh server challenge, which acts as the session key.
func (sb *SignalBridge) RegisterSession() string {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	c := GenerateChallenge()
	_, used := sb.sessions[c]
	if used {
		for used { // generate new challenges until we have an un-used one
			c = GenerateChallenge()
			_, used = sb.sessions[c]
		}
	}

	evtCh := make(chan Event)
	sb.sessions[c] = evtCh

	go func() { // make sure the session doesn't go stale and collect dust (ie unused memory)
		time.Sleep(10 * time.Minute)
		sb.mu.Lock()
		defer sb.mu.Unlock()
		delete(sb.sessions, c)
	}()

	return c
}

// GetEventChannel returns the channel for the passed challenge from which future events can be read.
// If sc doesn't exist, the 2nd argument is false.
func (sb *SignalBridge) GetEventChannel(sc string) (<-chan Event, bool) {
	sb.mu.Lock()
	defer sb.mu.Unlock()
	ch, has := sb.sessions[sc]
	return ch, has
}

// CompleteSession uses the passed challenge to send on and close the open channel.
// It will return an error if the session doesn't exist.
func (sb *SignalBridge) CompleteSession(sc string, success bool, token string) error {
	sb.mu.Lock()
	defer sb.mu.Unlock()

	ch, ok := sb.sessions[sc]
	if !ok {
		return fmt.Errorf("no such session")
	}

	var (
		err     error
		timeout = time.NewTimer(1 * time.Minute)

		evt = Event{
			Worked: success,
			Token:  token,
		}
	)

	// handle what happens if the sse client isn't connected
	select {
	case <-timeout.C:
		err = fmt.Errorf("faled to send completed session")

	case ch <- evt:
		timeout.Stop()
	}

	// session is finalized either way
	close(ch)
	delete(sb.sessions, sc)

	return err
}
