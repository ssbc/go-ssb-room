package go_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/muxrpc/handlers/tunnel/server"
)

// peers not on the members list can't connect
func TestStaleMembers(t *testing.T) {
	r := require.New(t)
	// a := assert.New(t)

	testPath := filepath.Join("testrun", t.Name())
	os.RemoveAll(testPath)

	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)
	t.Cleanup(cancel)

	// create the roomsrv
	ts := makeNamedTestBot(t, "server", ctx, nil)
	ctx = ts.ctx

	// two new clients, connected to server automaticaly
	// default to member during makeNamedTestBot
	tal, _ := ts.makeTestClient("tal")
	srh, srhFeed := ts.makeTestClient("srh")

	// announce srh so that other could connect
	var ok bool
	err := srh.Async(ctx, &ok, muxrpc.TypeJSON, muxrpc.Method{"room", "announce"})
	r.NoError(err)

	_, has := ts.srv.StateManager.Has(srhFeed)
	r.True(has, "srh should be connected")

	// shut down srh
	srh.Terminate()
	time.Sleep(1 * time.Second)
	_, has = ts.srv.StateManager.Has(srhFeed)
	r.False(has, "srh shouldn't be connected")

	// try to connect srh
	var arg server.ConnectArg
	arg.Portal = ts.srv.Whoami()
	arg.Target = srhFeed

	src, snk, err := tal.Duplex(ctx, muxrpc.TypeBinary, muxrpc.Method{"room", "connect"}, arg)
	r.NoError(err)

	// let server respond
	time.Sleep(1 * time.Second)

	// assert cant read
	r.False(src.Next(ctx), "source should be cancled")
	r.Error(src.Err(), "source should have an error")

	// assert cant write
	_, err = snk.Write([]byte("fake keyexchange"))
	r.Error(err, "stream should should be canceled")

	ts.srv.Shutdown()
	tal.Terminate()
	ts.srv.Close()

	// wait for all muxrpc serve()s to exit
	r.NoError(ts.serveGroup.Wait())
	cancel()

}
