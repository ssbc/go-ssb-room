// SPDX-License-Identifier: MIT

// Package alias implements the muxrpc handlers for alias needs.
package alias

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/aliases"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	refs "go.mindeco.de/ssb-refs"
)

// Handler implements the muxrpc methods for alias registration and recvocation
type Handler struct {
	logger kitlog.Logger
	self   refs.FeedRef

	db roomdb.AliasesService

	netInfo network.ServerEndpointDetails

	// roomDomain string // the http(s) domain of the room to signal alias addresses
}

// New returns a fresh alias muxrpc handler
func New(log kitlog.Logger, self refs.FeedRef, aliasesDB roomdb.AliasesService, netInfo network.ServerEndpointDetails) Handler {

	var h Handler
	h.self = self
	h.netInfo = netInfo
	h.logger = log
	h.db = aliasesDB

	return h
}

const sigSuffix = ".sig.ed25519"

var httpRouter = router.CompleteApp()

// Register is an async muxrpc method handler for registering aliases.
// It receives two string arguments over muxrpc (alias and signature),
// checks the signature confirmation is correct (for this room and signed by the key of theconnection)
// If it is valid, it registers the alias on the roomdb and returns true. If not it returns an error.
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
		var takenErr roomdb.ErrAliasTaken
		if errors.As(err, &takenErr) {
			return nil, takenErr
		}
		return nil, fmt.Errorf("registerAlias: could not register alias: %w", err)
	}

	return h.netInfo.URLForAlias(confirmation.Alias), nil
}

// Revoke checks that the alias is from that user before revoking the alias from the database.
func (h Handler) Revoke(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	var args []string

	err := json.Unmarshal(req.RawArgs, &args)
	if err != nil {
		return nil, fmt.Errorf("registerAlias: bad request: %w", err)
	}

	if n := len(args); n != 1 {
		return nil, fmt.Errorf("registerAlias: expected two arguments got %d", n)
	}

	// get the user from the muxrpc connection
	userID, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	alias, err := h.db.Resolve(ctx, args[0])
	if err != nil {
		return nil, err
	}

	if !alias.Feed.Equal(userID) {
		return nil, fmt.Errorf("revokeAlias: not your alias (moderators need to use the web dashboard of the room")
	}

	err = h.db.Revoke(ctx, alias.Name)
	if err != nil {
		return nil, err
	}

	return true, nil
}
