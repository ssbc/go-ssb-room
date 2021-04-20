// SPDX-License-Identifier: MIT

package go_test

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2/debug"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/testutils"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/repo"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomsrv"
)

var (
	initLogging sync.Once
	mainLog     = log.NewNopLogger()
)

// cant call testing.pkg in init()
func testInit(t *testing.T) {
	initLogging.Do(func() {
		if testing.Verbose() {
			mainLog = testutils.NewRelativeTimeLogger(nil)
		}
	})

	os.RemoveAll(filepath.Join("testrun", t.Name()))
}

type botServer struct {
	ctx context.Context
	log log.Logger
}

func newBotServer(ctx context.Context, log log.Logger) botServer {
	return botServer{ctx, log}
}

func (bs botServer) Serve(s *roomsrv.Server) func() error {
	return func() error {
		err := s.Network.Serve(bs.ctx)
		if err != nil {
			if errors.Is(err, io.EOF) || errors.Is(err, context.Canceled) {
				return nil
			}
			level.Warn(bs.log).Log("event", "bot serve exited", "err", err)
		}
		return err
	}
}

type testSession struct {
	t testing.TB

	srv *roomsrv.Server

	ctx        context.Context
	serveGroup *errgroup.Group
}

func makeNamedTestBot(t testing.TB, name string, ctx context.Context, opts []roomsrv.Option) *testSession {
	r := require.New(t)
	testPath := filepath.Join("testrun", t.Name(), "bot-"+name)
	os.RemoveAll(testPath)

	botOptions := append(opts,
		roomsrv.WithLogger(log.With(mainLog, "bot", name)),
		roomsrv.WithRepoPath(testPath),
		roomsrv.WithNetworkConnTracker(network.NewLastWinsTracker()),
		roomsrv.WithPostSecureConnWrapper(func(conn net.Conn) (net.Conn, error) {
			return debug.WrapDump(filepath.Join(testPath, "muxdump"), conn)
		}),
	)

	// could also use the mocks
	db, err := sqlite.Open(repo.New(testPath))
	r.NoError(err)
	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Log("db close failed: ", err)
		}
	})

	err = db.Config.SetPrivacyMode(context.TODO(), roomdb.ModeRestricted)
	r.NoError(err)

	netInfo := network.ServerEndpointDetails{
		Domain: name,

		ListenAddressMUXRPC: ":0",

		UseSubdomainForAliases: true,
	}

	sb := signinwithssb.NewSignalBridge()
	theBot, err := roomsrv.New(db.Members, db.DeniedKeys, db.Aliases, db.AuthWithSSB, sb, db.Config, netInfo, botOptions...)
	r.NoError(err)

	ts := testSession{
		t:   t,
		srv: theBot,
	}

	ts.serveGroup, ts.ctx = errgroup.WithContext(ctx)

	ts.serveGroup.Go(func() error {
		return theBot.Network.Serve(ts.ctx)
	})

	return &ts
}

// TODO: refactor for single test session and use makeTestClient()
func createServerAndBots(t *testing.T, ctx context.Context, count uint) []*testSession {
	testInit(t)
	r := require.New(t)

	appKey := make([]byte, 32)
	rand.Read(appKey)

	netOpts := []roomsrv.Option{
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
	}

	theBots := []*testSession{}

	session := makeNamedTestBot(t, "srv", ctx, netOpts)

	theBots = append(theBots, session)

	for i := uint(1); i < count+1; i++ {
		// TODO: replace with makeClient?!
		clientSession := makeNamedTestBot(t, fmt.Sprintf("%d", i), ctx, netOpts)
		theBots = append(theBots, clientSession)
	}

	t.Cleanup(func() {
		time.Sleep(1 * time.Second)
		for _, bot := range theBots {
			bot.srv.Shutdown()
			r.NoError(bot.srv.Close())
		}
	})

	return theBots
}
