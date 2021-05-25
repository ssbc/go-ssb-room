// SPDX-License-Identifier: MIT

package signinwithssb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"go.cryptoscope.co/muxrpc/v2"
	kitlog "go.mindeco.de/log"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	validate "github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	refs "go.mindeco.de/ssb-refs"
)

// Handler implements the muxrpc methods for the "Sign-in with SSB" calls. SendSolution and InvalidateAllSolutions.
type Handler struct {
	logger kitlog.Logger
	self   refs.FeedRef

	sessions roomdb.AuthWithSSBService
	members  roomdb.MembersService

	bridge *signinwithssb.SignalBridge

	roomDomain string // the http(s) domain of the room to signal redirect addresses
}

// New returns the muxrpc handler for Sign-in with SSB
func New(
	log kitlog.Logger,
	self refs.FeedRef,
	roomDomain string,
	membersdb roomdb.MembersService,
	sessiondb roomdb.AuthWithSSBService,
	bridge *signinwithssb.SignalBridge,
) Handler {

	var h Handler
	h.self = self
	h.roomDomain = roomDomain
	h.logger = log
	h.sessions = sessiondb
	h.members = membersdb
	h.bridge = bridge

	return h
}

// SendSolution implements the receiving end of httpAuth.sendSolution.
// It recevies three parameters [sc, cc, sol], does the validation and if it passes creates a token
// and signals the created token to the SSE HTTP handler using the signal bridge.
func (h Handler) SendSolution(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	clientID, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	member, err := h.members.GetByFeed(ctx, *clientID)
	if err != nil {
		return nil, fmt.Errorf("client is not a room member")
	}

	var params []string
	if err := json.Unmarshal(req.RawArgs, &params); err != nil {
		return nil, err
	}

	if n := len(params); n != 3 {
		return nil, fmt.Errorf("expected 3 arguments (sc, cc, sol) but got %d", n)
	}

	var payload validate.ClientPayload
	payload.ServerID = h.self
	payload.ServerChallenge = params[0]
	payload.ClientID = *clientID
	payload.ClientChallenge = params[1]

	sig, err := base64.StdEncoding.DecodeString(strings.TrimSuffix(params[2], ".sig.ed25519"))
	if err != nil {
		h.bridge.SessionFailed(payload.ServerChallenge, err)
		return nil, fmt.Errorf("signature is not valid base64 data: %w", err)
	}

	if !payload.Validate(sig) {
		err = fmt.Errorf("not a valid solution")
		h.bridge.SessionFailed(payload.ServerChallenge, err)
		return nil, err
	}

	tok, err := h.sessions.CreateToken(ctx, member.ID)
	if err != nil {
		h.bridge.SessionFailed(payload.ServerChallenge, err)
		return nil, err
	}

	err = h.bridge.SessionWorked(payload.ServerChallenge, tok)
	if err != nil {
		h.sessions.RemoveToken(ctx, tok)
		return nil, err
	}

	return true, nil
}

// InvalidateAllSolutions implements the muxrpc call httpAuth.invalidateAllSolutions
func (h Handler) InvalidateAllSolutions(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	// get the feed from the muxrpc connection
	clientID, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	// lookup the member
	member, err := h.members.GetByFeed(ctx, *clientID)
	if err != nil {
		return nil, err
	}

	// delete all SIWSSB sessions of that member
	err = h.sessions.WipeTokensForMember(ctx, member.ID)
	if err != nil {
		return nil, err
	}

	return true, nil
}
