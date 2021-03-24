package signinwithssb

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	kitlog "github.com/go-kit/kit/log"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	validate "github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	refs "go.mindeco.de/ssb-refs"
)

// Handler implements the muxrpc methods for alias registration and recvocation
type Handler struct {
	logger kitlog.Logger
	self   refs.FeedRef

	sessions roomdb.AuthWithSSBService
	members  roomdb.MembersService

	roomDomain string // the http(s) domain of the room to signal redirect addresses
}

// New returns a fresh alias muxrpc handler
func New(
	log kitlog.Logger,
	self refs.FeedRef,
	sessiondb roomdb.AuthWithSSBService,
	membersdb roomdb.MembersService,
	roomDomain string) Handler {

	var h Handler
	h.self = self
	h.roomDomain = roomDomain
	h.logger = log
	h.sessions = sessiondb
	h.members = membersdb

	return h
}

func (h Handler) SendSolution(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	clientID, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	var params []string
	if err := json.Unmarshal(req.RawArgs, &params); err != nil {
		return nil, err
	}

	if n := len(params); n != 3 {
		return nil, fmt.Errorf("expected 3 arguments (sc, cc, sol) but got %d", n)
	}

	var sol validate.ClientRequest
	sol.ServerID = h.self
	sol.ServerChallenge = params[0]
	sol.ClientID = *clientID
	sol.ClientChallenge = params[1]

	sig, err := base64.StdEncoding.DecodeString(params[2])
	if err != nil {
		return nil, fmt.Errorf("sc is not valid base64 data: %w", err)
	}

	if !sol.Validate(sig) {
		return nil, fmt.Errorf("not a valid solution")
	}

	// TODO:
	// h.challenges.Solved(sc)
	// return true, nil

	return nil, fmt.Errorf("TODO: update SSE")
}

func (h Handler) InvalidateAllSolutions(ctx context.Context, req *muxrpc.Request) (interface{}, error) {
	// get the feed from the muxrpc connection
	clientID, err := network.GetFeedRefFromAddr(req.RemoteAddr())
	if err != nil {
		return nil, err
	}

	member, err := h.members.GetByFeed(ctx, *clientID)
	if err != nil {
		return nil, err
	}

	err = h.sessions.WipeTokensForMember(ctx, member.ID)
	if err != nil {
		return nil, err
	}

	return true, nil
}
