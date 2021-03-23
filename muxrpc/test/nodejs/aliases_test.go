package nodejs_test

import (
	"testing"
	"time"

	"github.com/ssb-ngi-pointer/go-ssb-room/aliases"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/mockdb"
)

func TestGoServerJSClientAliases(t *testing.T) {
	a := assert.New(t)
	r := require.New(t)

	ts := newRandomSession(t)
	// ts := newSession(t, nil)

	var membersDB = &mockdb.FakeMembersService{}
	var aliasesDB = &mockdb.FakeAliasesService{}
	srv := ts.startGoServer(membersDB, aliasesDB)
	// allow all peers (there arent any we dont want to allow)
	membersDB.GetByFeedReturns(roomdb.Member{Nickname: "free4all"}, nil)

	alice := ts.startJSClient("alice", "./testscripts/modern_aliases.js",
		srv.Network.GetListenAddr(),
		srv.Whoami(),
	)

	time.Sleep(10 * time.Second)

	// wait for both to exit
	ts.wait()

	r.Equal(1, aliasesDB.RegisterCallCount(), "register call count")
	_, name, ref, signature := aliasesDB.RegisterArgsForCall(0)
	a.Equal("alice", name, "wrong alias registered")
	a.Equal(alice.Ref(), ref.Ref())

	var aliasReq aliases.Confirmation
	aliasReq.Alias = name
	aliasReq.Signature = signature
	aliasReq.UserID = alice
	aliasReq.RoomID = srv.Whoami()

	a.True(aliasReq.Verify(), "signature validation")

	r.Equal(1, aliasesDB.RevokeCallCount(), "revoke call count")
	_, name = aliasesDB.RevokeArgsForCall(0)
	a.Equal("alice", name, "wrong alias revoked")

}
