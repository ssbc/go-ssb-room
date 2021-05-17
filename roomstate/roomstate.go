package roomstate

import (
	"sort"
	"sync"

	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/broadcasts"
	refs "go.mindeco.de/ssb-refs"
)

type Manager struct {
	logger kitlog.Logger

	endpointsUpdater     broadcasts.EndpointsEmitter
	endpointsbroadcaster *broadcasts.EndpointsBroadcast

	attendantsUpdater     broadcasts.AttendantsEmitter
	attendantsbroadcaster *broadcasts.AttendantsBroadcast

	roomMu *sync.Mutex
	room   roomStateMap
}

func NewManager(log kitlog.Logger) *Manager {
	var m Manager
	m.logger = log
	m.endpointsUpdater, m.endpointsbroadcaster = broadcasts.NewEndpointsEmitter()
	m.attendantsUpdater, m.attendantsbroadcaster = broadcasts.NewAttendantsEmitter()
	m.roomMu = new(sync.Mutex)
	m.room = make(roomStateMap)

	return &m
}

// roomStateMap is a single room
type roomStateMap map[string]muxrpc.Endpoint

// copy map entries to list for broadcast update
func (rsm roomStateMap) AsList() []string {
	memberList := make([]string, 0, len(rsm))
	for m := range rsm {
		memberList = append(memberList, m)
	}
	sort.Strings(memberList)
	return memberList
}

func (m *Manager) RegisterLegacyEndpoints(sink broadcasts.EndpointsEmitter) {
	m.endpointsbroadcaster.Register(sink)
}

func (m *Manager) RegisterAttendantsUpdates(sink broadcasts.AttendantsEmitter) {
	m.attendantsbroadcaster.Register(sink)
}

// List just returns a list of feed references
func (m *Manager) List() []string {
	m.roomMu.Lock()
	defer m.roomMu.Unlock()
	return m.room.AsList()
}

// AddEndpoint adds the endpoint to the room
func (m *Manager) AddEndpoint(who refs.FeedRef, edp muxrpc.Endpoint) {
	m.roomMu.Lock()
	// add ref to to the room map
	m.room[who.Ref()] = edp
	// update all the connected tunnel.endpoints calls
	m.endpointsUpdater.Update(m.room.AsList())
	// update all the connected room.attendants calls
	m.attendantsUpdater.Joined(who)
	m.roomMu.Unlock()
}

// Remove removes the peer from the room
func (m *Manager) Remove(who refs.FeedRef) {
	m.roomMu.Lock()
	// remove ref from lobby
	delete(m.room, who.Ref())
	// update all the connected tunnel.endpoints calls
	m.endpointsUpdater.Update(m.room.AsList())
	// update all the connected room.attendants calls
	m.attendantsUpdater.Left(who)
	m.roomMu.Unlock()
}

// AlreadyAdded returns true if the peer was already added to the room.
// if it isn't it will be added.
func (m *Manager) AlreadyAdded(who refs.FeedRef, edp muxrpc.Endpoint) bool {
	m.roomMu.Lock()

	// if the peer didn't call tunnel.announce()
	_, has := m.room[who.Ref()]
	if !has {
		// register them as if they didnt
		m.room[who.Ref()] = edp

		// update everyone
		m.endpointsUpdater.Update(m.room.AsList())
		m.attendantsUpdater.Joined(who)
	}

	m.roomMu.Unlock()
	return has
}

// Has returns true and the endpoint if the peer is in the room
func (m *Manager) Has(who refs.FeedRef) (muxrpc.Endpoint, bool) {
	m.roomMu.Lock()
	// add ref to to the room map
	edp, has := m.room[who.Ref()]
	m.roomMu.Unlock()
	return edp, has
}
