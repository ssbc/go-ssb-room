package tunnel

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"go.mindeco.de/ssb-rooms/internal/network"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.cryptoscope.co/muxrpc/v2"
	"go.mindeco.de/ssb-rooms/internal/broadcasts"
)

type roomState struct {
	logger kitlog.Logger

	updater     broadcasts.RoomChangeSink
	broadcaster *broadcasts.RoomChangeBroadcast

	roomsMu sync.Mutex
	rooms   roomsStateMap
}

func (rs *roomState) stateTicker(ctx context.Context) {
	tick := time.NewTicker(1 * time.Second)
	for {
		select {
		case <-ctx.Done():
			tick.Stop()
			return

		case <-tick.C:
		}
		rs.roomsMu.Lock()
		for room, members := range rs.rooms {
			level.Info(rs.logger).Log("room", room, "cnt", len(members))
			for who := range members {
				level.Info(rs.logger).Log("room", room, "feed", who)
			}
		}
		rs.roomsMu.Unlock()
	}
}

// layout is map[room-name]map[canonical feedref]client-handle
type roomsStateMap map[string]roomStateMap

// roomStateMap is a single room
type roomStateMap map[string]muxrpc.Endpoint

func (rs *roomState) isRoom(context.Context, *muxrpc.Request) (interface{}, error) {
	level.Debug(rs.logger).Log("called", "isRoom")
	return true, nil
}

func (rs *roomState) ping(context.Context, *muxrpc.Request) (interface{}, error) {
	now := time.Now().UnixNano() / 1000
	level.Debug(rs.logger).Log("called", "ping")
	return now, nil
}

func (rs *roomState) announce(_ context.Context, req *muxrpc.Request) (interface{}, error) {
	level.Debug(rs.logger).Log("called", "announce")
	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	rs.roomsMu.Lock()
	rs.updater.Update(broadcasts.RoomChange{
		Op:  "joined",
		Who: *ref,
	})

	// add ref to lobby
	rs.rooms["lobby"][ref.Ref()] = req.Endpoint()
	members := len(rs.rooms["lobby"])
	rs.roomsMu.Unlock()

	return RoomUpdate{"joined", true, uint(members)}, nil
}

type RoomUpdate struct {
	Action  string
	Success bool
	Members uint
}

func (rs *roomState) leave(_ context.Context, req *muxrpc.Request) (interface{}, error) {
	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	rs.roomsMu.Lock()
	rs.updater.Update(broadcasts.RoomChange{
		Op:  "left",
		Who: *ref,
	})

	// add ref to lobby
	delete(rs.rooms["lobby"], ref.Ref())
	members := len(rs.rooms["lobby"])
	rs.roomsMu.Unlock()

	return RoomUpdate{"left", true, uint(members)}, nil
}

func (rs *roomState) endpoints(_ context.Context, req *muxrpc.Request, snk *muxrpc.ByteSink, edp muxrpc.Endpoint) error {
	level.Debug(rs.logger).Log("called", "endpoints")
	rs.broadcaster.Register(newForwarder(snk))
	return nil
}

type updateForwarder struct {
	snk *muxrpc.ByteSink
	enc *json.Encoder
}

func newForwarder(snk *muxrpc.ByteSink) updateForwarder {
	enc := json.NewEncoder(snk)
	snk.SetEncoding(muxrpc.TypeJSON)
	return updateForwarder{
		snk: snk,
		enc: enc,
	}
}

func (uf updateForwarder) Update(rc broadcasts.RoomChange) error {
	return uf.enc.Encode(rc)
}

func (uf updateForwarder) Close() error {
	return uf.snk.Close()
}
