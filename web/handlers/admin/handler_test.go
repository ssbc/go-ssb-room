package admin

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	refs "go.mindeco.de/ssb-refs"

	"github.com/PuerkitoBio/goquery"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
)

func TestDashoard(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminDashboard).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
		{"#roomCount", "AdminRoomCountPlural"},
	})
}

func TestAllowListEmpty(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminAllowListOverview).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "ListCountPlural"},
	})

}

func TestAllowList(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	lst := admindb.ListEntries{
		{ID: 1, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"}},
		{ID: 2, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: "test"}},
		{ID: 3, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: "true"}},
	}
	ts.AllowListDB.ListReturns(lst, nil)

	html, resp := ts.Client.GetHTML("/allow-list", nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "ListCountPlural"},
	})

	a.EqualValues(html.Find("#theList").Children().Length(), 3)

	lst = admindb.ListEntries{
		{ID: 666, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{1}, 32), Algo: "one"}},
	}
	ts.AllowListDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML("/allow-list", nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "ListCountPlural"}, // TODO: should be singular - template func testing stub might have a qurik
	})

	elems := html.Find("#theList").Children()
	a.EqualValues(elems.Length(), 1)

	// check for link to remove confirm link
	link, yes := elems.ContentsFiltered("a").Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/allow-list/remove/confirm?id=666", link)
}

// utils

type localizedElement struct {
	Selector, Label string
}

// we dont test for the text values, just the i18n placeholders
func assertLocalized(t *testing.T, html *goquery.Document, elems []localizedElement) {
	a := assert.New(t)
	for i, pair := range elems {
		a.Equal(pair.Label, html.Find(pair.Selector).Text(), "localized pair %d failed", i+1)
	}
}
