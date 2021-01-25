package tunnel

import (
	"context"
	"fmt"
	"time"

	kitlog "github.com/go-kit/kit/log"
	"github.com/go-kit/kit/log/level"
	"go.cryptoscope.co/muxrpc/v2"
)

type roomState struct {
	logger kitlog.Logger
}

func (rs roomState) isRoom(context.Context, *muxrpc.Request) (interface{}, error) {
	level.Debug(rs.logger).Log("called", "isRoom")
	return true, nil
}

func (rs roomState) ping(context.Context, *muxrpc.Request) (interface{}, error) {
	now := time.Now().UnixNano() / 1000
	level.Debug(rs.logger).Log("called", "ping")
	return now, nil
}

func (rs roomState) announce(context.Context, *muxrpc.Request) (interface{}, error) {
	level.Debug(rs.logger).Log("called", "announce")
	return nil, fmt.Errorf("TODO:announce")
}

func (rs roomState) leave(context.Context, *muxrpc.Request) (interface{}, error) {
	level.Debug(rs.logger).Log("called", "leave")
	return nil, fmt.Errorf("TODO:leave")
}

func (rs roomState) endpoints(context.Context, *muxrpc.Request, *muxrpc.ByteSink, muxrpc.Endpoint) error {
	level.Debug(rs.logger).Log("called", "endpoints")
	return fmt.Errorf("TODO:endpoints")
}
