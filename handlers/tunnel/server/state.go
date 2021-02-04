package server

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	refs "go.mindeco.de/ssb-refs"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/network"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.cryptoscope.co/muxrpc/v2"
	"github.com/ssb-ngi-pointer/gossb-rooms/internal/broadcasts"
)

type roomState struct {
	self refs.FeedRef

	logger kitlog.Logger

	updater     broadcasts.RoomChangeSink
	broadcaster *broadcasts.RoomChangeBroadcast

	roomsMu sync.Mutex
	rooms   roomsStateMap
}

func (rs *roomState) stateTicker(ctx context.Context) {
	tick := time.NewTicker(10 * time.Second)
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
				level.Info(rs.logger).Log("room", room, "feed", who[1:5])
			}
		}
		rs.roomsMu.Unlock()
	}
}

// layout is map[room-name]map[canonical feedref]client-handle
type roomsStateMap map[string]roomStateMap

// roomStateMap is a single room
type roomStateMap map[string]muxrpc.Endpoint

// copy map entries to list for broadcast update
func (rsm roomStateMap) asList() []string {
	memberList := make([]string, 0, len(rsm))
	for m := range rsm {
		memberList = append(memberList, m)
	}
	return memberList
}

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
	// add ref to lobby
	rs.rooms["lobby"][ref.Ref()] = req.Endpoint()
	// update all the connected tunnel.endpoints calls
	rs.updater.Update(rs.rooms["lobby"].asList())
	rs.roomsMu.Unlock()

	return false, nil
}

func (rs *roomState) leave(_ context.Context, req *muxrpc.Request) (interface{}, error) {
	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	rs.roomsMu.Lock()
	// remove ref from lobby
	delete(rs.rooms["lobby"], ref.Ref())
	// update all the connected tunnel.endpoints calls
	rs.updater.Update(rs.rooms["lobby"].asList())
	rs.roomsMu.Unlock()

	return false, nil
}

func (rs *roomState) endpoints(_ context.Context, req *muxrpc.Request, snk *muxrpc.ByteSink) error {
	level.Debug(rs.logger).Log("called", "endpoints")

	toPeer := newForwarder(snk)

	// for future updates
	rs.broadcaster.Register(toPeer)

	ref, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return err
	}

	rs.roomsMu.Lock()

	lobby := rs.rooms["lobby"]
	// if the peer didn't call tunnel.announce()
	if _, has := lobby[ref.Ref()]; has {
		// just send the current state to the new peer
		toPeer.Update(lobby.asList())
	} else {
		// register them as if they did
		lobby[ref.Ref()] = req.Endpoint()
		rs.rooms["lobby"] = lobby

		// update everyone
		rs.updater.Update(lobby.asList())
	}

	// send the current state
	rs.roomsMu.Unlock()

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

func (uf updateForwarder) Update(members []string) error {
	return uf.enc.Encode(members)
}

func (uf updateForwarder) Close() error {
	return uf.snk.Close()
}
