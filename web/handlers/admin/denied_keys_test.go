// SPDX-License-Identifier: MIT

package admin

import (
	"bytes"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	refs "go.mindeco.de/ssb-refs"
)

func TestDeniedKeysEmpty(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminDeniedKeysOverview).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDeniedKeysWelcome"},
		{"title", "AdminDeniedKeysTitle"},
		{"#DeniedKeysCount", "MemberCountPlural"},
	})
}

func TestDeniedKeysAdd(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	listURL, err := ts.Router.Get(router.AdminDeniedKeysOverview).URL()
	a.NoError(err)

	html, resp := ts.Client.GetHTML(listURL.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	formSelection := html.Find("form#add-entry")
	a.EqualValues(1, formSelection.Length())

	method, ok := formSelection.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := formSelection.Attr("action")
	a.True(ok, "form has action set")

	addURL, err := ts.Router.Get(router.AdminDeniedKeysAdd).URL()
	a.NoError(err)

	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, formSelection, []webassert.FormElement{
		{Name: "pub_key", Type: "text"},
		{Name: "comment", Type: "text"},
	})

	newKey := "@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519"
	addVals := url.Values{
		"comment": []string{"some comment"},
		// just any key that looks valid
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL.String(), addVals)
	a.Equal(http.StatusFound, rec.Code)

	a.Equal(1, ts.DeniedKeysDB.AddCallCount())
	_, addedKey, addedComment := ts.DeniedKeysDB.AddArgsForCall(0)
	a.Equal(newKey, addedKey.Ref())
	a.Equal("some comment", addedComment)
}

func TestDeniedKeysDontAddInvalid(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	r := require.New(t)

	addURL, err := ts.Router.Get(router.AdminDeniedKeysAdd).URL()
	a.NoError(err)

	newKey := "@some-garbage"
	addVals := url.Values{
		"comment": []string{"some-comment"},
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL.String(), addVals)
	a.Equal(http.StatusBadRequest, rec.Code)

	a.Equal(0, ts.DeniedKeysDB.AddCallCount())

	doc, err := goquery.NewDocumentFromReader(rec.Body)
	r.NoError(err)

	expErr := `bad request: feedRef: couldn't parse "@some-garbage"`
	gotMsg := doc.Find("#errBody").Text()
	if !a.True(strings.HasPrefix(gotMsg, expErr), "did not find errBody") {
		t.Log(gotMsg)
	}
}

func TestDeniedKeys(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	lst := []roomdb.ListEntry{
		{ID: 1, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"}},
		{ID: 2, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: "test"}},
		{ID: 3, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: "true"}},
	}
	ts.DeniedKeysDB.ListReturns(lst, nil)

	html, resp := ts.Client.GetHTML("/denied")
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDeniedKeysWelcome"},
		{"title", "AdminDeniedKeysTitle"},
		{"#DeniedKeysCount", "MemberCountPlural"},
	})

	a.EqualValues(html.Find("#theList li").Length(), 3)

	lst = []roomdb.ListEntry{
		{ID: 666, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{1}, 32), Algo: "one"}},
	}
	ts.DeniedKeysDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML("/denied")
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDeniedKeysWelcome"},
		{"title", "AdminDeniedKeysTitle"},
		{"#DeniedKeysCount", "MemberCountSingular"},
	})

	elems := html.Find("#theList li")
	a.EqualValues(elems.Length(), 1)

	// check for link to remove confirm link
	link, yes := elems.ContentsFiltered("a").Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/denied/remove/confirm?id=666", link)
}

func TestDeniedKeysRemoveConfirmation(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testKey, err := refs.ParseFeedRef("@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519")
	a.NoError(err)
	testEntry := roomdb.ListEntry{ID: 666, PubKey: *testKey}
	ts.DeniedKeysDB.GetByIDReturns(testEntry, nil)

	urlTo := web.NewURLTo(ts.Router)
	urlRemoveConfirm := urlTo(router.AdminDeniedKeysRemoveConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRemoveConfirm.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.Ref(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL, err := ts.Router.Get(router.AdminDeniedKeysRemove).URL()
	a.NoError(err)

	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "id", Type: "hidden", Value: "666"},
	})
}

func TestDeniedKeysRemove(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlTo := web.NewURLTo(ts.Router)
	urlRemove := urlTo(router.AdminDeniedKeysRemove)

	ts.DeniedKeysDB.RemoveIDReturns(nil)

	addVals := url.Values{"id": []string{"666"}}
	rec := ts.Client.PostForm(urlRemove.String(), addVals)
	a.Equal(http.StatusFound, rec.Code)

	a.Equal(1, ts.DeniedKeysDB.RemoveIDCallCount())
	_, theID := ts.DeniedKeysDB.RemoveIDArgsForCall(0)
	a.EqualValues(666, theID)

	// now for unknown ID
	ts.DeniedKeysDB.RemoveIDReturns(roomdb.ErrNotFound)
	addVals = url.Values{"id": []string{"667"}}
	rec = ts.Client.PostForm(urlRemove.String(), addVals)
	a.Equal(http.StatusNotFound, rec.Code)
	//TODO: update redirect code with flash errors
}
