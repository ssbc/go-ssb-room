package admin

import (
	"bytes"
	"net/http"
	"testing"

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
	// we dont test for the text values, just the i18n placeholders
	a.Equal(html.Find("#welcome").Text(), "AdminDashboardWelcome")
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

	// a.Equal(html.Find("h1").Text(), db[1].Name)
	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "AdminListCountPlural"},
	})

}

func TestAllowList(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	lst := []refs.FeedRef{
		{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"},
		{ID: bytes.Repeat([]byte("1312"), 8), Algo: "test"},
		{ID: bytes.Repeat([]byte("acab"), 8), Algo: "true"},
	}
	ts.AllowListDB.ListReturns(lst, nil)

	// url, err := router.News(nil).Get(router.NewsPost).URL("PostID", "1")
	// a.Nil(err)
	html, resp := ts.Client.GetHTML("/allow-list", nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	// a.Equal(html.Find("h1").Text(), db[1].Name)
	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "AdminListCountPlural"},
	})

	a.EqualValues(html.Find("#theList").Children().Length(), 3)

	lst = []refs.FeedRef{
		{ID: bytes.Repeat([]byte{1}, 32), Algo: "one"},
	}
	ts.AllowListDB.ListReturns(lst, nil)

	// url, err := router.News(nil).Get(router.NewsPost).URL("PostID", "1")
	// a.Nil(err)
	html, resp = ts.Client.GetHTML("/allow-list", nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	// a.Equal(html.Find("h1").Text(), db[1].Name)
	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminAllowListWelcome"},
		{"title", "AdminAllowListTitle"},
		{"#allowListCount", "AdminListCountPlural"},
	})

	a.EqualValues(html.Find("#theList").Children().Length(), 1)

}

// utils

type localizedElement struct {
	Selector, Label string
}

func assertLocalized(t *testing.T, html *goquery.Document, elems []localizedElement) {
	a := assert.New(t)
	for i, pair := range elems {
		a.Equal(pair.Label, html.Find(pair.Selector).Text(), "localized pair %d failed", i+1)
	}
}
