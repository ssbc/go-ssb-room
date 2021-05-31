package roomstate

import (
	"fmt"
	"sort"
	"sync"

	"go.cryptoscope.co/muxrpc/v2"
	kitlog "go.mindeco.de/log"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/internal/broadcasts"
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

// List just returns a list of feed references as strings
func (m *Manager) List() []string {
	m.roomMu.Lock()
	defer m.roomMu.Unlock()
	return m.room.AsList()
}

func (m *Manager) ListAsRefs() []refs.FeedRef {
	m.roomMu.Lock()
	lst := m.room.AsList()
	m.roomMu.Unlock()

	rlst := make([]refs.FeedRef, len(lst))
	for i, s := range lst {
		fr, err := refs.ParseFeedRef(s)
		if err != nil {
			panic(fmt.Errorf("invalid feed ref in room state: %d: %s", i, err))
		}
		rlst[i] = *fr
	}
	return rlst
}

// AddEndpoint adds the endpoint to the room
func (m *Manager) AddEndpoint(who refs.FeedRef, edp muxrpc.Endpoint) {
	m.roomMu.Lock()
	// add ref to to the room map
	m.room[who.Ref()] = edp
	currentMembers := m.room.AsList()
	m.roomMu.Unlock()
	// update all the connected tunnel.endpoints calls
	m.endpointsUpdater.Update(currentMembers)
	// update all the connected room.attendants calls
	m.attendantsUpdater.Joined(who)
}

// Remove removes the peer from the room
func (m *Manager) Remove(who refs.FeedRef) {
	m.roomMu.Lock()
	// remove ref from lobby
	delete(m.room, who.Ref())
	currentMembers := m.room.AsList()
	m.roomMu.Unlock()
	// update all the connected tunnel.endpoints calls
	m.endpointsUpdater.Update(currentMembers)
	// update all the connected room.attendants calls
	m.attendantsUpdater.Left(who)
}

// AlreadyAdded returns true if the peer was already added to the room.
// if it isn't it will be added.
func (m *Manager) AlreadyAdded(who refs.FeedRef, edp muxrpc.Endpoint) bool {
	m.roomMu.Lock()

	var currentMembers []string
	// if the peer didn't call tunnel.announce()
	_, has := m.room[who.Ref()]
	if !has {
		// register them as if they didnt
		m.room[who.Ref()] = edp
		currentMembers = m.room.AsList()
	}
	m.roomMu.Unlock()

	if !has {
		// update everyone
		m.endpointsUpdater.Update(currentMembers)
		m.attendantsUpdater.Joined(who)
	}

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
