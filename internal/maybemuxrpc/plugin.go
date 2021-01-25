package maybemuxrpc

import (
	"net"
	"sync"

	"go.cryptoscope.co/muxrpc/v2"
)

type Authorizer interface {
	Authorize(net.Conn) bool
}

type Plugin interface {
	// Name returns the name and version of the plugin.
	// format: name-1.0.2
	Name() string

	// Method returns the preferred method of the call
	Method() muxrpc.Method

	// Handler returns the muxrpc handler for the plugin
	Handler() muxrpc.Handler

	Authorizer
}

type PluginManager interface {
	Register(Plugin)
	MakeHandler(conn net.Conn) (muxrpc.Handler, error)
}

type pluginManager struct {
	regLock sync.Mutex // protects the map
	plugins map[string]Plugin
}

func NewPluginManager() PluginManager {
	return &pluginManager{
		plugins: make(map[string]Plugin),
	}
}

func (pmgr *pluginManager) Register(p Plugin) {
	//  access race
	pmgr.regLock.Lock()
	defer pmgr.regLock.Unlock()
	pmgr.plugins[p.Method().String()] = p
}

func (pmgr *pluginManager) MakeHandler(conn net.Conn) (muxrpc.Handler, error) {

	pmgr.regLock.Lock()
	defer pmgr.regLock.Unlock()

	h := muxrpc.HandlerMux{}

	for _, p := range pmgr.plugins {
		if !p.Authorize(conn) {
			continue
		}

		h.Register(p.Method(), p.Handler())
	}

	return &h, nil
}
