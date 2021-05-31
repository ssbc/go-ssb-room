// SPDX-License-Identifier: MIT

package go_test

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/muxrpc/handlers/tunnel/server"
)

// peers not on the members list can't connect
func TestStaleMembers(t *testing.T) {
	testInit(t)

	r := require.New(t)

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
	tal := ts.makeTestClient("tal") // https://en.wikipedia.org/wiki/Tal_(name)
	srh := ts.makeTestClient("srh") // https://en.wikipedia.org/wiki/Sarah

	// announce srh so that tal could connect

	_, has := ts.srv.StateManager.Has(srh.feed)

	// shut down srh
	srh.Terminate()
	time.Sleep(1 * time.Second)
	_, has = ts.srv.StateManager.Has(srh.feed)
	r.False(has, "srh shouldn't be connected")

	// try to connect srh
	var arg server.ConnectArg
	arg.Portal = ts.srv.Whoami()
	arg.Target = srh.feed

	src, snk, err := tal.Duplex(ctx, muxrpc.TypeBinary, muxrpc.Method{"room", "connect"}, arg)
	r.NoError(err)

	time.Sleep(1 * time.Second) // let server respond

	// assert cant read
	r.False(src.Next(ctx), "source should be cancled")
	r.Error(src.Err(), "source should have an error")

	// assert cant write
	testKexMsg := []byte("fake keyexchange")
	_, err = snk.Write(testKexMsg)
	r.Error(err, "stream should should be canceled")

	// restart srh
	oldSrh := srh.feed
	srh = ts.makeTestClient("srh")
	r.True(oldSrh.Equal(&srh.feed))
	t.Log("restarted srh")

	time.Sleep(1 * time.Second) // let server respond

	// announce srh so that tal can connect
	var ok bool
	err = srh.Async(ctx, &ok, muxrpc.TypeJSON, muxrpc.Method{"tunnel", "announce"})
	r.NoError(err)
	r.True(ok)
	t.Log("announced srh again")

	testKexReply := []byte("fake kex reply")

	// prepare srh for incoming call
	receivedCall := make(chan struct{})
	receivedKexMessage := make(chan struct{})
	srh.mockedHandler.HandledCalls(func(m muxrpc.Method) bool { return m.String() == "tunnel.connect" })
	srh.mockedHandler.HandleCallCalls(func(ctx context.Context, req *muxrpc.Request) {
		m := req.Method.String()
		t.Log("received call: ", m)
		if m != "tunnel.connect" {
			return
		}
		t.Log("correct call received")
		close(receivedCall)

		// receive a msg
		src, err := req.ResponseSource()
		if err != nil {
			panic(fmt.Errorf("expected source for duplex call: %s", err))
		}

		if !src.Next(ctx) {
			err = fmt.Errorf("did not get message from source: %v", src.Err())
			panic(err)
		}

		gotKexMsg, err := src.Bytes()
		if err != nil {
			panic(err)
		}

		if !bytes.Equal(testKexMsg, gotKexMsg) {
			panic(fmt.Sprintf("wrong kex message: %q", gotKexMsg))
		}

		close(receivedKexMessage)

		// send a msg
		snk, err := req.ResponseSink()
		if err != nil {
			panic(fmt.Errorf("expected sink for duplex call: %s", err))
		}
		snk.Write(testKexReply)
	})

	// 2nd try to connect (should work)
	src, snk, err = tal.Duplex(ctx, muxrpc.TypeBinary, muxrpc.Method{"tunnel", "connect"}, arg)
	r.NoError(err)

	<-receivedCall

	_, err = snk.Write(testKexMsg)
	r.NoError(err, "stream should should be canceled")

	<-receivedKexMessage

	// assert can read
	r.True(src.Next(ctx), "source should be cancled")
	r.NoError(src.Err(), "source should have an error")

	gotKexMsgFromSrh, err := src.Bytes()
	r.NoError(err)
	r.Equal(testKexReply, gotKexMsgFromSrh)

	// shut everythign down
	ts.srv.Shutdown()
	tal.Terminate()
	srh.Terminate()
	ts.srv.Close()

	// wait for all muxrpc serve()s to exit
	r.NoError(ts.serveGroup.Wait())
	cancel()

}
