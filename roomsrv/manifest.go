package roomsrv

import (
	"context"
	"encoding/json"
	"fmt"
	"net"

	"go.cryptoscope.co/muxrpc/v2"
)

type namedPlugin struct {
	h    muxrpc.Handler
	name string
}

func (np namedPlugin) Name() string            { return np.name }
func (np namedPlugin) Method() muxrpc.Method   { return muxrpc.Method{np.name} }
func (np namedPlugin) Handler() muxrpc.Handler { return np.h }
func (np namedPlugin) Authorize(net.Conn) bool { return true }

type manifestHandler string

func (manifestHandler) Handled(m muxrpc.Method) bool { return m.String() == "manifest" }

func (manifestHandler) HandleConnect(context.Context, muxrpc.Endpoint) {}

func (h manifestHandler) HandleCall(ctx context.Context, req *muxrpc.Request) {
	err := req.Return(ctx, json.RawMessage(h))
	if err != nil {
		fmt.Println("manifest err", err)
	}
}

func init() {
	if !json.Valid([]byte(manifest)) {
		manifestMap := make(map[string]interface{})
		err := json.Unmarshal([]byte(manifest), &manifestMap)
		fmt.Println(err)
		panic("manifest blob is broken json")
	}
}

// this is a very simple hardcoded manifest.json dump which oasis' ssb-client expects to do it's magic.
const manifest manifestHandler = `
{
	"manifest": "sync",

	"whoami":"async",

	"tunnel": {
		"announce": "sync",
		"leave": "sync",
		"connect": "duplex",
		"endpoints": "source",
		"isRoom": "async",
		"ping": "sync"
	}
}`

var manifestPlug = namedPlugin{
	h:    manifest,
	name: "manifest",
}
