// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package sqlite

import (
	"bytes"
	"context"
	"crypto/rand"
	"os"
	"path/filepath"
	"testing"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/internal/repo"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/stretchr/testify/require"
)

func TestFallbackAuth(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)
	tr := repo.New(testRepo)

	// fake feed for testing, looks ok at least
	newMember := refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: refs.RefAlgoFeedSSB1}

	db, err := Open(tr)
	r.NoError(err, "failed to open database")

	memberID, err := db.Members.Add(ctx, newMember, roomdb.RoleMember)
	r.NoError(err, "failed to create member")

	testPassword := "super-secure-and-secret-password"

	err = db.AuthFallback.SetPassword(ctx, memberID, testPassword)
	r.NoError(err, "failed to create password")

	cookieVal, err := db.AuthFallback.Check(newMember.Ref(), string(testPassword))
	r.NoError(err, "failed to check password")
	gotID, ok := cookieVal.(int64)
	r.True(ok, "unexpected cookie value: %T", cookieVal)
	r.Equal(memberID, gotID, "unexpected member ID value")

	// now check we can also use an alias
	testAliasLogin := "test-alias-login"

	// 64 bytes of random for testing (validation is handled by the handlers)
	testSig := make([]byte, 64)
	rand.Read(testSig)

	err = db.Aliases.Register(ctx, testAliasLogin, newMember, testSig)
	r.NoError(err, "failed to register the test alias")

	cookieVal2, err := db.AuthFallback.Check(testAliasLogin, string(testPassword))
	r.NoError(err, "failed to check password via alias")
	gotIDforAlias, ok := cookieVal2.(int64)
	r.True(ok, "unexpected cookie value: %T", cookieVal)
	r.Equal(memberID, gotIDforAlias, "unexpected member ID value")

	r.NoError(db.Close())
}

func TestFallbackAuthSetPassword(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)
	tr := repo.New(testRepo)

	// fake feed for testing, looks ok at least
	newMember := refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: refs.RefAlgoFeedSSB1}

	db, err := Open(tr)
	r.NoError(err, "failed to open database")

	memberID, err := db.Members.Add(ctx, newMember, roomdb.RoleMember)
	r.NoError(err, "failed to create member")

	testPassword := "super-secure-and-secret-password"

	err = db.AuthFallback.SetPassword(ctx, memberID, testPassword)
	r.NoError(err, "failed to set password")

	// use the password
	cookieVal, err := db.AuthFallback.Check(newMember.Ref(), string(testPassword))
	r.NoError(err, "failed to check password")
	gotID, ok := cookieVal.(int64)
	r.True(ok, "unexpected cookie value: %T", cookieVal)
	r.Equal(memberID, gotID, "unexpected member ID value")

	// use a wrong password
	cookieVal, err = db.AuthFallback.Check(newMember.Ref(), string(testPassword)+"nope-nope-nope")
	r.Error(err, "wrong password actually worked?!")
	r.Nil(cookieVal)

	// set it to something different
	changedTestPassword := "some-different-super-secure-password"
	err = db.AuthFallback.SetPassword(ctx, memberID, changedTestPassword)
	r.NoError(err, "failed to update password")

	// now try to use old and new
	cookieVal, err = db.AuthFallback.Check(newMember.Ref(), string(testPassword))
	r.Error(err, "old password actually worked?!")
	r.Nil(cookieVal)

	cookieVal, err = db.AuthFallback.Check(newMember.Ref(), string(changedTestPassword))
	r.NoError(err, "new password didnt work")
	gotID, ok = cookieVal.(int64)
	r.True(ok, "unexpected cookie value: %T", cookieVal)
	r.Equal(memberID, gotID, "unexpected member ID value")
}

func TestFallbackAuthSetPasswordWithToken(t *testing.T) {
	r := require.New(t)
	ctx := context.Background()

	testRepo := filepath.Join("testrun", t.Name())
	os.RemoveAll(testRepo)
	tr := repo.New(testRepo)

	// two fake feeds for testing, looks ok at least
	alf := refs.FeedRef{ID: bytes.Repeat([]byte("whyy"), 8), Algo: refs.RefAlgoFeedSSB1}
	carl := refs.FeedRef{ID: bytes.Repeat([]byte("carl"), 8), Algo: refs.RefAlgoFeedSSB1}

	db, err := Open(tr)
	r.NoError(err, "failed to open database")

	alfID, err := db.Members.Add(ctx, alf, roomdb.RoleModerator)
	r.NoError(err, "failed to create member")

	carlID, err := db.Members.Add(ctx, carl, roomdb.RoleModerator)
	r.NoError(err, "failed to create member")

	err = db.AuthFallback.SetPassword(ctx, carlID, "i swear i wont forgettt thiszzz91238129e812hjejahsdkasdhaksjdh")
	r.NoError(err, "failed to update password")

	// and he does... so lets create a token for him
	resetTok, err := db.AuthFallback.CreateResetToken(ctx, alfID, carlID)
	r.NoError(err)

	// has to be a from valid user tho
	noToken, err := db.AuthFallback.CreateResetToken(ctx, 666, carlID)
	r.Error(err)
	r.Equal("", noToken)

	// change carls password by using the token
	newPassword := "marry had a little lamp"
	err = db.AuthFallback.SetPasswordWithToken(ctx, resetTok, newPassword)
	r.NoError(err, "setPassword with token failed")

	// now use the new password
	cookieVal, err := db.AuthFallback.Check(carl.Ref(), newPassword)
	r.NoError(err, "new password didnt work")
	gotID, ok := cookieVal.(int64)
	r.True(ok, "unexpected cookie value: %T", cookieVal)
	r.Equal(carlID, gotID, "unexpected member ID value")
}
