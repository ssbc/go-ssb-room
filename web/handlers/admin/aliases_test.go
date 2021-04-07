package admin

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
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

	urlTo := web.NewURLTo(ts.Router)
	urlRevokeConfirm := urlTo(router.AdminAliasesRevokeConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRevokeConfirm.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.Ref(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL, err := ts.Router.Get(router.AdminAliasesRevoke).URL()
	a.NoError(err)

	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "name", Type: "hidden", Value: testEntry.Name},
	})
}

func TestAliasesRevoke(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlTo := web.NewURLTo(ts.Router)
	urlRevoke := urlTo(router.AdminAliasesRevoke)

	ts.AliasesDB.RevokeReturns(nil)

	addVals := url.Values{"name": []string{"the-name"}}
	rec := ts.Client.PostForm(urlRevoke.String(), addVals)
	a.Equal(http.StatusFound, rec.Code)

	a.Equal(1, ts.AliasesDB.RevokeCallCount())
	_, theName := ts.AliasesDB.RevokeArgsForCall(0)
	a.EqualValues("the-name", theName)

	// now for unknown ID
	ts.AliasesDB.RevokeReturns(roomdb.ErrNotFound)
	addVals = url.Values{"name": []string{"nope"}}
	rec = ts.Client.PostForm(urlRevoke.String(), addVals)
	a.Equal(http.StatusNotFound, rec.Code)
	//TODO: update redirect code with flash errors
}
