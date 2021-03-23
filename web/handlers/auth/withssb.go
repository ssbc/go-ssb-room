// SPDX-License-Identifier: MIT

package auth

import (
	"context"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"go.cryptoscope.co/muxrpc/v2"
	"go.mindeco.de/http/auth"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/network"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	refs "go.mindeco.de/ssb-refs"
)

// withssbHandler implements the oauth-like challenge/response dance described in
// https://ssb-ngi-pointer.github.io/rooms2/#sign-in-with-ssb
type withssbHandler struct {
	roomID refs.FeedRef

	members roomdb.MembersService
	aliases roomdb.AliasesService

	cookieAuth *auth.Handler

	endpoints network.Endpoints
}

func (h withssbHandler) login(w http.ResponseWriter, req *http.Request) (interface{}, error) {
	queryParams := req.URL.Query()

	var clientReq signinwithssb.ClientRequest
	clientReq.ServerID = h.roomID // fll inthe server

	// validate and update client challange
	cc := queryParams.Get("challenge")
	if _, err := signinwithssb.DecodeChallengeString(cc); err != nil {
		return nil, weberrors.ErrBadRequest{Where: "client-challange", Details: err}
	}
	clientReq.ClientChallange = cc

	// check who the client is
	var client refs.FeedRef
	if cid := queryParams.Get("cid"); cid != "" {
		parsed, err := refs.ParseFeedRef(cid)
		if err != nil {
			return nil, weberrors.ErrBadRequest{Where: "cid", Details: err}
		}
		client = *parsed
	} else {
		alias, err := h.aliases.Resolve(req.Context(), queryParams.Get("alias"))
		if err != nil {
			return nil, weberrors.ErrBadRequest{Where: "alias", Details: err}
		}
		client = alias.Feed
	}

	// check that we have that member
	member, err := h.members.GetByFeed(req.Context(), client)
	if err != nil {
		if err == roomdb.ErrNotFound {
			return nil, weberrors.ErrForbidden{Details: fmt.Errorf("sign-in: client isnt a member")}
		}
		return nil, err
	}
	clientReq.ClientID = client

	// get the connected client for that member
	edp, connected := h.endpoints.GetEndpointFor(client)
	if !connected {
		return nil, weberrors.ErrForbidden{Details: fmt.Errorf("sign-in: client not connected to room")}
	}

	// roll a challange from the server
	sc := signinwithssb.GenerateChallenge()
	clientReq.ServerChallange = sc

	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()

	// request the signed solution over muxrpc
	var solution string
	err = edp.Async(ctx, &solution, muxrpc.TypeString, muxrpc.Method{"httpAuth", "requestSolution"}, sc, cc)
	if err != nil {
		return nil, err
	}

	// decode and validate the response
	solutionBytes, err := base64.URLEncoding.DecodeString(solution)
	if err != nil {
		return nil, err
	}

	if !clientReq.Validate(solutionBytes) {
		return nil, fmt.Errorf("sign-in with ssb: validation of client solution failed")
	}

	// create a cookie for the member
	// TODO: pass in session store
	// TODO: revamp auth check (check different cookie fields, don't reuse password session)
	// err = h.cookieAuth.SaveUserSession(req, w, member.ID)
	// if err != nil {
	// 	return nil, err
	// }

	// TODO: store the solution for session invalidation
	// https://github.com/ssb-ngi-pointer/go-ssb-room/issues/92

	return "you are now logged in!", nil
}
