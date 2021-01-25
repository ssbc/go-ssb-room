package go_test

import (
	"context"
	"errors"
	"io"

	"github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.mindeco.de/ssb-rooms/roomsrv"
)

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
