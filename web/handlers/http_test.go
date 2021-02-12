// SPDX-License-Identifier: MIT

package handlers

import (
	"bytes"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

func TestIndex(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)
	r := require.New(t)

	url, err := ts.Router.Get(router.CompleteIndex).URL()
	r.Nil(err)
	html, resp := ts.Client.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	assertLocalized(t, html, []localizedElement{
		{"h1", "LandingWelcome"},
		{"title", "LandingTitle"},
		// {"#nav", "FooBar"},
	})

	val, has := html.Find("img").Attr("src")
	a.True(has, "logo src attribute not found")
	a.Equal("/assets/img/test-hermie.png", val)
}

func TestAbout(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)
	r := require.New(t)

	url, err := ts.Router.Get(router.CompleteAbout).URL()
	r.Nil(err)
	html, resp := ts.Client.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("The about page", found)
}

func TestNotFound(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	html, resp := ts.Client.GetHTML("/some/random/ASDKLANZXC", nil)
	a.Equal(http.StatusNotFound, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #404 - Not Found", found)
}

func TestNewsRegisterd(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	html, resp := ts.Client.GetHTML("/news/", nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	assertLocalized(t, html, []localizedElement{
		{"#welcome", "NewsWelcome"},
		{"title", "NewsTitle"},
	})
}

func TestRestricted(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	testURLs := []string{
		// "/admin/",
		"/admin/admin",
		"/admin/admin/",
	}

	for _, turl := range testURLs {
		html, resp := ts.Client.GetHTML(turl, nil)
		a.Equal(http.StatusUnauthorized, resp.Code, "wrong HTTP status code for %q", turl)
		found := html.Find("h1").Text()
		a.Equal("Error #401 - Unauthorized", found, "wrong error message code for %q", turl)
	}
}

func TestLoginForm(t *testing.T) {
	ts := setup(t)

	a, r := assert.New(t), require.New(t)

	url, err := ts.Router.Get(router.AuthFallbackSignInForm).URL()
	r.Nil(err)
	html, resp := ts.Client.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AuthFallbackWelcome"},
		{"title", "AuthFallbackTitle"},
	})
}

func TestFallbackAuth(t *testing.T) {
	ts := setup(t)

	a, r := assert.New(t), require.New(t)

	loginVals := url.Values{
		"user": []string{"test"},
		"pass": []string{"test"},
	}
	ts.AuthFallbackDB.CheckReturns(int64(23), nil)

	url, err := ts.Router.Get(router.AuthFallbackSignIn).URL()
	r.Nil(err)
	url.Host = "localhost"
	url.Scheme = "http"

	resp := ts.Client.PostForm(url.String(), loginVals)
	a.Equal(http.StatusSeeOther, resp.Code, "wrong HTTP status code for sign in")

	a.Equal(1, ts.AuthFallbackDB.CheckCallCount())

	// very cheap client session
	jar, err := cookiejar.New(nil)
	r.NoError(err)

	c := resp.Result().Cookies()
	jar.SetCookies(url, c)

	var h = http.Header(map[string][]string{})

	dashboardURL, err := ts.Router.Get(router.AdminDashboard).URL()
	r.Nil(err)
	dashboardURL.Host = "localhost"
	dashboardURL.Scheme = "http"

	cs := jar.Cookies(dashboardURL)
	r.Len(cs, 1, "expecting one cookie!")
	theCookie := cs[0].String()
	a.NotEqual("", theCookie, "should have a new cookie")
	h.Set("Cookie", theCookie)
	// t.Log(h)
	// durl := dashboardURL.String()
	durl := "http://localhost/admin"
	// durl := "/admin"
	t.Log(durl)
	html, resp := ts.Client.GetHTML(durl, &h)
	if !a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for dashboard") {
		t.Log(html.Find("body").Text())
	}

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
	})

	testRef := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{0}, 16)}
	ts.RoomState.AddEndpoint(testRef, nil)

	html, resp = ts.Client.GetHTML(durl, &h)
	if !a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code") {
		t.Log(html.Find("body").Text())
	}

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
		{"#roomCount", "AdminRoomCountSingular"},
	})

	testRef2 := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{1}, 16)}
	ts.RoomState.AddEndpoint(testRef2, nil)

	html, resp = ts.Client.GetHTML(durl, &h)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	t.Log(html.Find("body").Text())
	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
		{"#roomCount", "AdminRoomCountPlural"},
	})

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
