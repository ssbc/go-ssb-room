package broadcasts

import (
	"io"
	"sync"

	"github.com/hashicorp/go-multierror"
	refs "go.mindeco.de/ssb-refs"
)

type RoomChange struct {
	Op  string
	Who refs.FeedRef
}

type RoomChangeSink interface {
	Update(value RoomChange) error
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
func (bcst *broadcastSink) Update(rc RoomChange) error {

	bcst.mu.Lock()
	for s := range bcst.sinks {
		err := (*s).Update(rc)
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
		wg   sync.WaitGroup
		merr *multierror.Error
	)

	// might be fine without the waitgroup and concurrency

	wg.Add(len(sinks))
	for _, sink_ := range sinks {
		go func(sink RoomChangeSink) {
			defer wg.Done()

			err := sink.Close()
			if err != nil {
				merr = multierror.Append(merr, err)
				return
			}
		}(sink_)
	}
	wg.Wait()

	return merr.ErrorOrNil()
}
