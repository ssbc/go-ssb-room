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

func TestAllowListEmpty(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminAllowListOverview).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "MemberCountPlural"},
	})
}

func TestAllowListAdd(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	listURL, err := ts.Router.Get(router.AdminAllowListOverview).URL()
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

	addURL, err := ts.Router.Get(router.AdminAllowListAdd).URL()
	a.NoError(err)

	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, formSelection, []webassert.FormElement{
        {Tag: "input", Name: "pub_key", Type: "text"},
	})

	newKey := "@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519"
	addVals := url.Values{
		// just any key that looks valid
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL.String(), addVals)
	a.Equal(http.StatusFound, rec.Code)

	a.Equal(1, ts.AllowListDB.AddCallCount())
	_, added := ts.AllowListDB.AddArgsForCall(0)
	a.Equal(newKey, added.Ref())
}

func TestAllowListDontAddInvalid(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	r := require.New(t)

	addURL, err := ts.Router.Get(router.AdminAllowListAdd).URL()
	a.NoError(err)

	newKey := "@some-garbage"
	addVals := url.Values{
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL.String(), addVals)
	a.Equal(http.StatusBadRequest, rec.Code)

	a.Equal(0, ts.AllowListDB.AddCallCount())

	doc, err := goquery.NewDocumentFromReader(rec.Body)
	r.NoError(err)

	expErr := `bad request: feedRef: couldn't parse "@some-garbage"`
	gotMsg := doc.Find("#errBody").Text()
	if !a.True(strings.HasPrefix(gotMsg, expErr), "did not find errBody") {
		t.Log(gotMsg)
	}
}

func TestAllowList(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	lst := roomdb.ListEntries{
		{ID: 1, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"}},
		{ID: 2, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: "test"}},
		{ID: 3, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: "true"}},
	}
	ts.AllowListDB.ListReturns(lst, nil)

	html, resp := ts.Client.GetHTML("/members")
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "MemberCountPlural"},
	})

	a.EqualValues(html.Find("#theList li").Length(), 3)

	lst = roomdb.ListEntries{
		{ID: 666, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{1}, 32), Algo: "one"}},
	}
	ts.AllowListDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML("/members")
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "MemberCountSingular"},
	})

	elems := html.Find("#theList li")
	a.EqualValues(elems.Length(), 1)

	// check for link to remove confirm link
	link, yes := elems.ContentsFiltered("a").Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/members/remove/confirm?id=666", link)
}

func TestAllowListRemoveConfirmation(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testKey, err := refs.ParseFeedRef("@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519")
	a.NoError(err)
	testEntry := roomdb.ListEntry{ID: 666, PubKey: *testKey}
	ts.AllowListDB.GetByIDReturns(testEntry, nil)

	urlTo := web.NewURLTo(ts.Router)
	urlRemoveConfirm := urlTo(router.AdminAllowListRemoveConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRemoveConfirm.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.Ref(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL, err := ts.Router.Get(router.AdminAllowListRemove).URL()
	a.NoError(err)

	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
        {Tag: "input", Name: "id", Type: "hidden", Value: "666"},
	})
}

func TestAllowListRemove(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlTo := web.NewURLTo(ts.Router)
	urlRemove := urlTo(router.AdminAllowListRemove)

	ts.AllowListDB.RemoveIDReturns(nil)

	addVals := url.Values{"id": []string{"666"}}
	rec := ts.Client.PostForm(urlRemove.String(), addVals)
	a.Equal(http.StatusFound, rec.Code)

	a.Equal(1, ts.AllowListDB.RemoveIDCallCount())
	_, theID := ts.AllowListDB.RemoveIDArgsForCall(0)
	a.EqualValues(666, theID)

	// now for unknown ID
	ts.AllowListDB.RemoveIDReturns(roomdb.ErrNotFound)
	addVals = url.Values{"id": []string{"667"}}
	rec = ts.Client.PostForm(urlRemove.String(), addVals)
	a.Equal(http.StatusNotFound, rec.Code)
	//TODO: update redirect code with flash errors
}
