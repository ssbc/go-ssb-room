// SPDX-License-Identifier: MIT

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

func TestDeniedKeysEmpty(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url := ts.URLTo(router.AdminDeniedKeysOverview)

	html, resp := ts.Client.GetHTML(url)
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

	listURL := ts.URLTo(router.AdminDeniedKeysOverview)

	html, resp := ts.Client.GetHTML(listURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	formSelection := html.Find("form#add-entry")
	a.EqualValues(1, formSelection.Length())

	method, ok := formSelection.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := formSelection.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminDeniedKeysAdd)
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
	rec := ts.Client.PostForm(addURL, addVals)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)

	a.Equal(1, ts.DeniedKeysDB.AddCallCount())
	_, addedKey, addedComment := ts.DeniedKeysDB.AddArgsForCall(0)
	a.Equal(newKey, addedKey.Ref())
	a.Equal("some comment", addedComment)
}

func TestDeniedKeysDontAddInvalid(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	addURL := ts.URLTo(router.AdminDeniedKeysAdd)

	newKey := "@some-garbage"
	addVals := url.Values{
		"comment": []string{"some-comment"},
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL, addVals)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)

	a.Equal(0, ts.DeniedKeysDB.AddCallCount(), "did not call add")

	listURL := ts.URLTo(router.AdminDeniedKeysOverview)
	res := rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	webassert.HasFlashMessages(t, ts.Client, listURL, "ErrorBadRequest")
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

	deniedOverviewURL := ts.URLTo(router.AdminDeniedKeysOverview)

	html, resp := ts.Client.GetHTML(deniedOverviewURL)
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

	html, resp = ts.Client.GetHTML(deniedOverviewURL)
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
	wantLink := ts.URLTo(router.AdminDeniedKeysRemoveConfirm, "id", 666)
	a.Equal(wantLink.String(), link)
}

func TestDeniedKeysRemoveConfirmation(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testKey, err := refs.ParseFeedRef("@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519")
	a.NoError(err)
	testEntry := roomdb.ListEntry{ID: 666, PubKey: *testKey}
	ts.DeniedKeysDB.GetByIDReturns(testEntry, nil)

	urlRemoveConfirm := ts.URLTo(router.AdminDeniedKeysRemoveConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRemoveConfirm)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.Ref(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminDeniedKeysRemove)
	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "id", Type: "hidden", Value: "666"},
	})
}

func TestDeniedKeysRemove(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlRemove := ts.URLTo(router.AdminDeniedKeysRemove)

	ts.DeniedKeysDB.RemoveIDReturns(nil)

	addVals := url.Values{"id": []string{"666"}}
	rec := ts.Client.PostForm(urlRemove, addVals)
	a.Equal(http.StatusFound, rec.Code)

	a.Equal(1, ts.DeniedKeysDB.RemoveIDCallCount())
	_, theID := ts.DeniedKeysDB.RemoveIDArgsForCall(0)
	a.EqualValues(666, theID)

	// now for unknown ID
	ts.DeniedKeysDB.RemoveIDReturns(roomdb.ErrNotFound)
	addVals = url.Values{"id": []string{"667"}}
	rec = ts.Client.PostForm(urlRemove, addVals)
	a.Equal(http.StatusNotFound, rec.Code)
	//TODO: update redirect code with flash errors
}
