// SPDX-License-Identifier: MIT

package alias

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/aliases"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	refs "go.mindeco.de/ssb-refs"
)

type Handler struct {
	logger kitlog.Logger
	self   refs.FeedRef

	db roomdb.AliasService
}

const sigSuffix = ".sig.ed25519"

func (h Handler) Register(ctx context.Context, req *muxrpc.Request) (interface{}, error) {

	var args []string

	err := json.Unmarshal(req.RawArgs, &args)
	if err != nil {
		return nil, fmt.Errorf("registerAlias: bad request: %w", err)
	}

	if n := len(args); n != 2 {
		return nil, fmt.Errorf("registerAlias: expected two arguments got %d", n)
	}

	if !strings.HasSuffix(args[1], sigSuffix) {
		return nil, fmt.Errorf("registerAlias: signature does not have the expected suffix")
	}

	// remove the suffix of the base64 string
	sig := strings.TrimSuffix(args[1], sigSuffix)

	var confirmation aliases.Confirmation
	confirmation.RoomID = h.self
	confirmation.Alias = args[0]
	confirmation.Signature, err = base64.StdEncoding.DecodeString(sig)
	if err != nil {
		return nil, fmt.Errorf("registerAlias: bad signature encoding: %w", err)
	}

	// check alias is valid
	if !aliases.IsValid(confirmation.Alias) {
		return nil, fmt.Errorf("registerAlias: invalid alias")
	}

	// get the user from the muxrpc connection
	userID, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	confirmation.UserID = *userID

	// check the signature
	if !confirmation.Verify() {
		return nil, fmt.Errorf("registerAlias: invalid signature")
	}

	err = h.db.Register(ctx, confirmation.Alias, confirmation.UserID, confirmation.Signature)
	if err != nil {
		return nil, fmt.Errorf("registerAlias: could not register alias: %w", err)
	}

	return true, nil
}
