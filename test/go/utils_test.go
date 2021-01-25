package go_test

import (
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"github.com/stretchr/testify/require"

	"go.mindeco.de/ssb-rooms/internal/maybemod/testutils"
	"go.mindeco.de/ssb-rooms/internal/network"
	"go.mindeco.de/ssb-rooms/roomsrv"
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

func makeNamedTestBot(t testing.TB, name string, opts []roomsrv.Option) *roomsrv.Server {
	r := require.New(t)
	testPath := filepath.Join("testrun", t.Name(), "bot-"+name)

	botOptions := append(opts,
		roomsrv.WithLogger(log.With(mainLog, "bot", name)),
		roomsrv.WithRepoPath(testPath),
		roomsrv.WithListenAddr(":0"),
		roomsrv.WithNetworkConnTracker(network.NewLastWinsTracker()),
	)

	theBot, err := roomsrv.New(botOptions...)
	r.NoError(err)
	return theBot
}
