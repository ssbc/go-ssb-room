package admin

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	refs "go.mindeco.de/ssb-refs"
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

	testAlias := roomdb.Alias{
		Name: "testAlias",
		Feed: ts.User.PubKey,
	}
	ts.AliasesDB.ResolveReturns(testAlias, nil)

	ts.AliasesDB.RevokeReturns(nil)

	revokeVal := url.Values{"name": []string{testAlias.Name}}
	rec := ts.Client.PostForm(urlRevoke, revokeVal)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)
	a.Equal(overviewURL.Path, rec.Header().Get("Location"))
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, overviewURL, "AdminMemberDetailsAliasRevoked")

	a.Equal(1, ts.AliasesDB.RevokeCallCount())
	_, theName := ts.AliasesDB.RevokeArgsForCall(0)
	a.EqualValues(testAlias.Name, theName)

	// now for unknown ID
	ts.AliasesDB.RevokeReturns(roomdb.ErrNotFound)
	revokeVal = url.Values{"name": []string{"nope"}}
	rec = ts.Client.PostForm(urlRevoke, revokeVal)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)
	a.Equal(overviewURL.Path, rec.Header().Get("Location"))
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, overviewURL, "ErrorNotFound")
}

func TestAliasesRevokeByMember(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	// the alias we test against
	testAlias := roomdb.Alias{
		Name: "testalias",
		Feed: refs.FeedRef{
			ID:   bytes.Repeat([]byte("1"), 32),
			Algo: "ed25519",
		},
	}
	ts.AliasesDB.ResolveReturns(testAlias, nil)

	// start with someone else trying to remove it
	ts.User = roomdb.Member{
		ID: 666,
		PubKey: refs.FeedRef{
			ID:   bytes.Repeat([]byte("2"), 32),
			Algo: "ed25519",
		},
		Role: roomdb.RoleMember,
	}

	urlRevoke := ts.URLTo(router.AdminAliasesRevoke)
	overviewURL := ts.URLTo(router.AdminMembersOverview)

	revokeVal := url.Values{"name": []string{testAlias.Name}}
	rec := ts.Client.PostForm(urlRevoke, revokeVal)
	a.Equal(http.StatusForbidden, rec.Code)
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, overviewURL, "ErrorForbidden")
	a.Equal(0, ts.AliasesDB.RevokeCallCount())

	// now change the user to the one owning the alias
	ts.User = roomdb.Member{
		ID:     123,
		PubKey: testAlias.Feed,
		Role:   roomdb.RoleMember,
	}

	rec = ts.Client.PostForm(urlRevoke, revokeVal)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)
	a.Equal(overviewURL.Path, rec.Header().Get("Location"))
	a.True(len(rec.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, overviewURL, "AdminMemberDetailsAliasRevoked")

	if a.Equal(1, ts.AliasesDB.RevokeCallCount()) {
		_, theName := ts.AliasesDB.RevokeArgsForCall(0)
		a.EqualValues(testAlias.Name, theName)
	}

}
