// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web/router"
	"github.com/ssbc/go-ssb-room/v2/web/webassert"
	refs "github.com/ssbc/go-ssb-refs"
)

func TestAliasesRevokeConfirmation(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testKey, err := refs.ParseFeedRef("@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519")
	a.NoError(err)
	testEntry := roomdb.Alias{ID: 666, Name: "the-test-name", Feed: *testKey}
	ts.AliasesDB.GetByIDReturns(testEntry, nil)

	urlRevokeConfirm := ts.URLTo(router.AdminAliasesRevokeConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRevokeConfirm)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.Ref(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminAliasesRevoke)
	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "name", Type: "hidden", Value: testEntry.Name},
	})
}

func TestAliasesRevoke(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlRevoke := ts.URLTo(router.AdminAliasesRevoke)
	overviewURL := ts.URLTo(router.AdminMembersOverview)

	aliasEntry := roomdb.Alias{
		ID:   ts.User.ID,
		Feed: ts.User.PubKey,
		Name: "Blobby",
	}
	ts.AliasesDB.RevokeReturns(nil)
	ts.AliasesDB.ResolveReturns(aliasEntry, nil)

	addVals := url.Values{"name": []string{"the-name"}}
	rec := ts.Client.PostForm(urlRevoke, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)
	a.Equal(overviewURL.Path, rec.Header().Get("Location"))
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, overviewURL, "AdminMemberDetailsAliasRevoked")

	a.Equal(1, ts.AliasesDB.RevokeCallCount())
	_, theName := ts.AliasesDB.RevokeArgsForCall(0)
	a.EqualValues("the-name", theName)

	// now for unknown ID
	ts.AliasesDB.RevokeReturns(roomdb.ErrNotFound)
	addVals = url.Values{"name": []string{"nope"}}
	rec = ts.Client.PostForm(urlRevoke, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)
	a.Equal(overviewURL.Path, rec.Header().Get("Location"))
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, overviewURL, "ErrorNotFound")
}
