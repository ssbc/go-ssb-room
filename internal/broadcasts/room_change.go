package broadcasts

import (
	"io"
	"sync"

	"go.mindeco.de/ssb-rooms/internal/maybemod/multierror"
)

type RoomChangeSink interface {
	Update(members []string) error
	io.Closer
}

// NewRoomChanger returns the Sink, to write to the broadcaster, and the new
// broadcast instance.
func NewRoomChanger() (RoomChangeSink, *RoomChangeBroadcast) {
	bcst := RoomChangeBroadcast{
		mu:    &sync.Mutex{},
		sinks: make(map[*RoomChangeSink]struct{}),
	}

	return (*broadcastSink)(&bcst), &bcst
}

// RoomChangeBroadcast is an interface for registering one or more Sinks to recieve
// updates.
type RoomChangeBroadcast struct {
	mu    *sync.Mutex
	sinks map[*RoomChangeSink]struct{}
}

// Register a Sink for updates to be sent. also returns
func (bcst *RoomChangeBroadcast) Register(sink RoomChangeSink) func() {
	bcst.mu.Lock()
	defer bcst.mu.Unlock()
	bcst.sinks[&sink] = struct{}{}

	return func() {
		bcst.mu.Lock()
		defer bcst.mu.Unlock()
		delete(bcst.sinks, &sink)
		sink.Close()
	}
}

type broadcastSink RoomChangeBroadcast

// Pour implements the Sink interface.
func (bcst *broadcastSink) Update(members []string) error {

	bcst.mu.Lock()
	for s := range bcst.sinks {
		err := (*s).Update(members)
		if err != nil {
			delete(bcst.sinks, s)
		}
	}
	bcst.mu.Unlock()

	return nil
}

// Close implements the Sink interface.
func (bcst *broadcastSink) Close() error {
	var sinks []RoomChangeSink

	bcst.mu.Lock()
	defer bcst.mu.Unlock()

	sinks = make([]RoomChangeSink, 0, len(bcst.sinks))

	for sink := range bcst.sinks {
		sinks = append(sinks, *sink)
	}

	var (
		wg sync.WaitGroup
		me multierror.List
	)

	// might be fine without the waitgroup and concurrency

	wg.Add(len(sinks))
	for _, sink_ := range sinks {
		go func(sink RoomChangeSink) {
			defer wg.Done()

			err := sink.Close()
			if err != nil {
				me.Errs = append(me.Errs, err)
				return
			}
		}(sink_)
	}
	wg.Wait()

	if len(me.Errs) == 0 {
		return nil
	}

	return me
}
