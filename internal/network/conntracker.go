// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package network

import (
	"context"
	"log"
	"net"
	"sync"
	"time"

	"github.com/ssbc/go-netwrap"
	"github.com/ssbc/go-secretstream"
)

type connEntry struct {
	c       net.Conn
	started time.Time
	done    chan struct{}
	cancel  context.CancelFunc
}

type connLookupMap map[[32]byte]connEntry

func toActive(a net.Addr) [32]byte {
	var pk [32]byte
	shs, ok := netwrap.GetAddr(a, "shs-bs").(secretstream.Addr)
	if !ok {
		panic("not an SHS connection")
	}
	copy(pk[:], shs.PubKey)
	return pk
}

func NewConnTracker() ConnTracker {
	return &connTracker{active: make(connLookupMap)}
}

// tracks open connections and refuses to established pubkeys
type connTracker struct {
	activeLock sync.Mutex
	active     connLookupMap
}

func (ct *connTracker) CloseAll() {
	ct.activeLock.Lock()
	defer ct.activeLock.Unlock()
	for k, c := range ct.active {
		if err := c.c.Close(); err != nil {
			log.Printf("failed to close %x: %v\n", k[:5], err)
		}
		c.cancel()
		// seems nice but we are holding the lock
		// <-c.done
		// delete(ct.active, k)
		// we must _trust_ the connection is hooked up to OnClose to remove it's entry
	}
}

func (ct *connTracker) Count() uint {
	ct.activeLock.Lock()
	defer ct.activeLock.Unlock()
	return uint(len(ct.active))
}

func (ct *connTracker) Active(a net.Addr) (bool, time.Duration) {
	ct.activeLock.Lock()
	defer ct.activeLock.Unlock()
	k := toActive(a)
	l, ok := ct.active[k]
	if !ok {
		return false, 0
	}
	return true, time.Since(l.started)
}

func (ct *connTracker) OnAccept(ctx context.Context, conn net.Conn) (bool, context.Context) {
	ct.activeLock.Lock()
	defer ct.activeLock.Unlock()
	k := toActive(conn.RemoteAddr())
	_, ok := ct.active[k]
	if ok {
		return false, nil
	}
	ctx, cancel := context.WithCancel(ctx)
	ct.active[k] = connEntry{
		c:       conn,
		started: time.Now(),
		done:    make(chan struct{}),
		cancel:  cancel,
	}
	return true, ctx
}

func (ct *connTracker) OnClose(conn net.Conn) time.Duration {
	ct.activeLock.Lock()
	defer ct.activeLock.Unlock()

	k := toActive(conn.RemoteAddr())
	who, ok := ct.active[k]
	if !ok {
		return 0
	}
	close(who.done)
	delete(ct.active, k)
	return time.Since(who.started)
}

// NewLastWinsTracker returns a conntracker that just kills the previous connection and let's the new one in.
func NewLastWinsTracker() ConnTracker {
	return &trackerLastWins{connTracker{active: make(connLookupMap)}}
}

type trackerLastWins struct {
	connTracker
}

func (ct *trackerLastWins) OnAccept(ctx context.Context, newConn net.Conn) (bool, context.Context) {
	ct.activeLock.Lock()
	k := toActive(newConn.RemoteAddr())
	oldConn, ok := ct.active[k]
	ct.activeLock.Unlock()
	if ok {
		oldConn.c.Close()
		oldConn.cancel()
		select {
		case <-oldConn.done:
			// cleaned up after itself
		case <-time.After(10 * time.Second):
			log.Println("[ConnTracker/lastWins] warning: not accepted, would ghost connection:", oldConn.c.RemoteAddr().String(), time.Since(oldConn.started))
			return false, nil
		}
	}
	ct.activeLock.Lock()
	ctx, cancel := context.WithCancel(ctx)
	ct.active[k] = connEntry{
		c:       newConn,
		started: time.Now(),
		done:    make(chan struct{}),
		cancel:  cancel,
	}
	ct.activeLock.Unlock()
	return true, ctx
}
