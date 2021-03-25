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
// It returns a channel from which future events can be read
// and the server challenge, which acts as the session key.
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

	ch <- Event{
		Worked: success,
		Token:  token,
	}
	close(ch)

	// remove session
	delete(sb.sessions, sc)

	return nil
}
