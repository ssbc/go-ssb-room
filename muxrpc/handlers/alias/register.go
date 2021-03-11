package alias

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/ssb-ngi-pointer/go-ssb-room/aliases"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"

	kitlog "github.com/go-kit/kit/log"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"go.cryptoscope.co/muxrpc/v2"
	refs "go.mindeco.de/ssb-refs"
)

type Handler struct {
	logger kitlog.Logger
	self   refs.FeedRef

	db roomdb.AliasService
}

func (h Handler) Register(ctx context.Context, req *muxrpc.Request) (interface{}, error) {

	var args []json.RawMessage

	err := json.Unmarshal(req.RawArgs, &args)
	if err != nil {
		return nil, fmt.Errorf("registerAlias: bad request: %w", err)
	}

	if n := len(args); n != 2 {
		return nil, fmt.Errorf("registerAlias: expected two arguments got %d", n)
	}

	var confirmation aliases.Confirmation
	confirmation.RoomID = h.self
	confirmation.Alias = string(args[0])
	// check alias is valid
	// if !aliases.IsValid(confirmation.Alias) { ... }

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

	err = h.db.Register(ctx, confirmation.Alias, confirmation.Signature)
	if err != nil {
		return nil, fmt.Errorf("registerAlias: could not register alias: %w", err)
	}

	return true, nil
}
