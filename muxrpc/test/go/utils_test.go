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

func makeNamedTestBot(t testing.TB, name string, opts []roomsrv.Option) (roomdb.MembersService, *roomsrv.Server) {
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
	return db.Members, theBot
}

type testBot struct {
	Server  *roomsrv.Server
	Members roomdb.MembersService
}

func createServerAndBots(t *testing.T, ctx context.Context, count uint) []testBot {
	testInit(t)
	r := require.New(t)

	botgroup, ctx := errgroup.WithContext(ctx)

	bs := newBotServer(ctx, mainLog)

	appKey := make([]byte, 32)
	rand.Read(appKey)

	netOpts := []roomsrv.Option{
		roomsrv.WithAppKey(appKey),
		roomsrv.WithContext(ctx),
	}
	theBots := []testBot{}

	srvsMembers, serv := makeNamedTestBot(t, "srv", netOpts)
	botgroup.Go(bs.Serve(serv))
	theBots = append(theBots, testBot{
		Server:  serv,
		Members: srvsMembers,
	})

	for i := uint(1); i < count+1; i++ {
		botMembers, botSrv := makeNamedTestBot(t, fmt.Sprintf("%d", i), netOpts)
		botgroup.Go(bs.Serve(botSrv))
		theBots = append(theBots, testBot{
			Server:  botSrv,
			Members: botMembers,
		})
	}

	t.Cleanup(func() {
		time.Sleep(1 * time.Second)
		for _, bot := range theBots {
			bot.Server.Shutdown()
			r.NoError(bot.Server.Close())
		}
		r.NoError(botgroup.Wait())
	})

	return theBots
}
