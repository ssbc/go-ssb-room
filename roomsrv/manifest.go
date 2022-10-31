// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package roomsrv

import (
	"context"
	"encoding/json"
	"fmt"

	"go.cryptoscope.co/muxrpc/v2"
)

type manifestHandler string

func (h manifestHandler) HandleAsync(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	return json.RawMessage(h), nil
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

	"gossip": {
		"ping": "duplex"
	},

	"room": {
		"registerAlias": "async",
		"revokeAlias": "async",
		"listAliases": "async",

		"connect": "duplex",
		"attendants": "source",
		"members": "source",
		"metadata": "async",
		"ping": "sync"
	},

	"tunnel": {
		"announce": "sync",
		"leave": "sync",
		"connect": "duplex",
		"endpoints": "source",
		"isRoom": "async",
		"ping": "sync"
	}
}`
