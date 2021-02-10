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
	setup(t)
	t.Cleanup(teardown)

	a := assert.New(t)
	r := require.New(t)

	url, err := testRouter.Get(router.CompleteIndex).URL()
	r.Nil(err)
	html, resp := testClient.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	assertLocalized(t, html, []localizedElement{
		{"#welcome", "LandingWelcome"},
		{"title", "LandingTitle"},
		// {"#nav", "FooBar"},
	})

	val, has := html.Find("#logo").Attr("src")
	a.True(has, "logo src attribute not found")
	a.Equal("/assets/img/test-hermie.png", val)
}

func TestAbout(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)
	r := require.New(t)

	url, err := testRouter.Get(router.CompleteAbout).URL()
	r.Nil(err)
	html, resp := testClient.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("The about page", found)
}

func TestNotFound(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)

	html, resp := testClient.GetHTML("/some/random/ASDKLANZXC", nil)
	a.Equal(http.StatusNotFound, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #404 - Not Found", found)
}

func TestNewsRegisterd(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)

	html, resp := testClient.GetHTML("/news/", nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	assertLocalized(t, html, []localizedElement{
		{"#welcome", "NewsWelcome"},
		{"title", "NewsTitle"},
	})
}

func TestRestricted(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)

	html, resp := testClient.GetHTML("/admin/", nil)
	a.Equal(http.StatusUnauthorized, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #401 - Unauthorized", found)
}

func TestLoginForm(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a, r := assert.New(t), require.New(t)

	url, err := testRouter.Get(router.AuthFallbackSignInForm).URL()
	r.Nil(err)
	html, resp := testClient.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AuthFallbackWelcome"},
		{"title", "AuthFallbackTitle"},
	})
}

func TestFallbackAuth(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a, r := assert.New(t), require.New(t)

	loginVals := url.Values{
		"user": []string{"test"},
		"pass": []string{"test"},
	}
	testAuthFallbackDB.CheckReturns(int64(23), nil)

	url, err := testRouter.Get(router.AuthFallbackSignInForm).URL()
	r.Nil(err)
	url.Host = "localhost"
	url.Scheme = "http"

	resp := testClient.PostForm(url.String(), loginVals)
	a.Equal(http.StatusSeeOther, resp.Code, "wrong HTTP status code")

	a.Equal(1, testAuthFallbackDB.CheckCallCount())

	// very cheap client session
	jar, err := cookiejar.New(nil)
	r.NoError(err)

	c := resp.Result().Cookies()
	jar.SetCookies(url, c)

	var h = http.Header(map[string][]string{})

	dashboardURL, err := testRouter.Get(router.AdminDashboard).URL()
	r.Nil(err)
	dashboardURL.Host = "localhost"
	dashboardURL.Scheme = "http"

	cs := jar.Cookies(dashboardURL)
	r.Len(cs, 1, "expecting one cookie!")
	theCookie := cs[0].String()
	a.NotEqual("", theCookie, "should have a new cookie")
	h.Set("Cookie", theCookie)

	html, resp := testClient.GetHTML(dashboardURL.String(), &h)
	if !a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code") {
		t.Log(html.Find("body").Text())
	}

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
	})

	testRef := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{0}, 16)}
	testRoomState.AddEndpoint(testRef, nil)

	html, resp = testClient.GetHTML(dashboardURL.String(), &h)
	if !a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code") {
		t.Log(html.Find("body").Text())
	}

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
		{"#roomCount", "AdminRoomCountSingular"},
	})

	testRef2 := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{1}, 16)}
	testRoomState.AddEndpoint(testRef2, nil)

	html, resp = testClient.GetHTML(dashboardURL.String(), &h)
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
