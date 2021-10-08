// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package whoami

import (
	"context"

	"go.cryptoscope.co/muxrpc/v2/typemux"

	"go.cryptoscope.co/muxrpc/v2"
	kitlog "go.mindeco.de/log"
	"go.mindeco.de/log/level"

	refs "go.mindeco.de/ssb-refs"
)

var (
	method = muxrpc.Method{"whoami"}
)

func checkAndLog(log kitlog.Logger, err error) {
	if err != nil {
		level.Warn(log).Log("event", "faild to write panic file", "err", err)
	}
}

func New(id refs.FeedRef) typemux.AsyncHandler {
	return handler{id: id}
}

type handler struct {
	id refs.FeedRef
}

func (h handler) HandleAsync(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	type ret struct {
		ID string `json:"id"`
	}

	return ret{h.id.Ref()}, nil
}
