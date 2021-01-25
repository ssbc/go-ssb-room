package go_test

import (
	"context"
	"crypto/rand"
	"net"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2/debug"
	"golang.org/x/sync/errgroup"

	"go.mindeco.de/ssb-rooms/internal/maybemod/multicloser/testutils"
	"go.mindeco.de/ssb-rooms/roomsrv"
)

func TestTunnelServerSimple(t *testing.T) {
	// defer leakcheck.Check(t)
	r := require.New(t)
	if testing.Short() {
		return
	}
	ctx, cancel := context.WithCancel(context.TODO())
	botgroup, ctx := errgroup.WithContext(ctx)

	info := testutils.NewRelativeTimeLogger(nil)
	bs := newBotServer(ctx, info)

	os.RemoveAll("testrun")

	appKey := make([]byte, 32)
	rand.Read(appKey)
	hmacKey := make([]byte, 32)
	rand.Read(hmacKey)

	srvLog := log.With(info, "peer", "srv")
	srv, err := roomsrv.New(
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
		roomsrv.WithLogger(srvLog),
		roomsrv.WithPostSecureConnWrapper(func(conn net.Conn) (net.Conn, error) {
			return debug.WrapDump(filepath.Join("testrun", t.Name(), "muxdump"), conn)
		}),
		roomsrv.WithRepoPath(filepath.Join("testrun", t.Name(), "srv")),
		roomsrv.WithListenAddr(":0"),
	)
	r.NoError(err)
	botgroup.Go(bs.Serve(srv))

	bobLog := log.With(info, "peer", "bob")
	bob, err := roomsrv.New(
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
		roomsrv.WithLogger(bobLog),
		roomsrv.WithRepoPath(filepath.Join("testrun", t.Name(), "bob")),
		roomsrv.WithListenAddr(":0"),
	)
	r.NoError(err)
	botgroup.Go(bs.Serve(bob))

	// TODO
	// srv.Replicate(bob.KeyPair.Id)
	// bob.Replicate(srv.KeyPair.Id)

	sess := &simpleSession{
		ctx: ctx,
		srv: srv,
		bob: bob,
		redial: func(t *testing.T) {
			t.Log("noop")
		},
	}

	tests := []struct {
		name string
		tf   func(t *testing.T)
	}{
		{"empty", sess.simple},
		// {"justMe", sess.wantFirst},
		// {"eachOne", sess.eachOne},
		// {"eachOneConnet", sess.eachOneConnet},
		// {"eachOneBothWant", sess.eachOnBothWant},
	}

	// all on a single connection
	r.NoError(err)
	for _, tc := range tests {
		t.Run("noop/"+tc.name, tc.tf)
	}

	srv.Shutdown()
	bob.Shutdown()
	cancel()

	r.NoError(srv.Close())
	r.NoError(bob.Close())
	r.NoError(botgroup.Wait())
}

type simpleSession struct {
	ctx context.Context

	redial func(t *testing.T)

	srv, bob *roomsrv.Server
}
