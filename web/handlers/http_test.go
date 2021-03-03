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
	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	assertLocalized(t, html, []localizedElement{
		{"h1", "Default Notice Title"},
		{"title", "Default Notice Title"},
		// {"#nav", "FooBar"},
	})

	content := html.Find("p").Text()
	a.Equal("Default Notice Content", content)
}

func TestAbout(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)
	r := require.New(t)

	url, err := ts.Router.Get(router.CompleteAbout).URL()
	r.Nil(err)
	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("The about page", found)
}

func TestNotFound(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	html, resp := ts.Client.GetHTML("/some/random/ASDKLANZXC")
	a.Equal(http.StatusNotFound, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #404 - Not Found", found)
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
		html, resp := ts.Client.GetHTML(turl)
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
	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AuthFallbackWelcome"},
		{"title", "AuthFallbackTitle"},
	})
}

func TestFallbackAuth(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	// very cheap client session
	jar, err := cookiejar.New(nil)
	r.NoError(err)

	signInFormURL, err := ts.Router.Get(router.AuthFallbackSignInForm).URL()
	r.Nil(err)
	signInFormURL.Host = "localhost"
	signInFormURL.Scheme = "https"

	doc, resp := ts.Client.GetHTML(signInFormURL.String())
	a.Equal(http.StatusOK, resp.Code)

	assertCSRFTokenPresent(t, doc.Find("form"))

	csrfCookie := resp.Result().Cookies()
	a.Len(csrfCookie, 1, "should have one cookie for CSRF protection validation")
	t.Log(csrfCookie)
	jar.SetCookies(signInFormURL, csrfCookie)

	csrfTokenElem := doc.Find("input[type=hidden]")
	a.Equal(1, csrfTokenElem.Length())

	csrfName, has := csrfTokenElem.Attr("name")
	a.True(has, "should have a name attribute")

	csrfValue, has := csrfTokenElem.Attr("value")
	a.True(has, "should have value attribute")

	loginVals := url.Values{
		"user": []string{"test"},
		"pass": []string{"test"},

		csrfName: []string{csrfValue},
	}
	ts.AuthFallbackDB.CheckReturns(int64(23), nil)

	t.Log(loginVals)

	signInURL, err := ts.Router.Get(router.AuthFallbackSignIn).URL()
	r.Nil(err)

	signInURL.Host = "localhost"
	signInURL.Scheme = "https"

	var csrfCookieHeader = http.Header(map[string][]string{})
	csrfCookieHeader.Set("Referer", "https://localhost")
	cs := jar.Cookies(signInURL)
	r.Len(cs, 1, "expecting one cookie for csrf")
	theCookie := cs[0].String()
	a.NotEqual("", theCookie, "should have a new cookie")
	csrfCookieHeader.Set("Cookie", theCookie)
	ts.Client.SetHeaders(csrfCookieHeader)

	resp = ts.Client.PostForm(signInURL.String(), loginVals)
	a.Equal(http.StatusSeeOther, resp.Code, "wrong HTTP status code for sign in")

	a.Equal(1, ts.AuthFallbackDB.CheckCallCount())

	sessionCookie := resp.Result().Cookies()
	jar.SetCookies(signInURL, sessionCookie)

	dashboardURL, err := ts.Router.Get(router.AdminDashboard).URL()
	r.Nil(err)
	dashboardURL.Host = "localhost"
	dashboardURL.Scheme = "https"

	var sessionHeader = http.Header(map[string][]string{})
	cs = jar.Cookies(dashboardURL)
	// TODO: why doesnt this return the csrf cookie?!
	// r.Len(cs, 2, "expecting one cookie!")
	for _, c := range cs {
		theCookie := c.String()
		a.NotEqual("", theCookie, "should have a new cookie")
		sessionHeader.Add("Cookie", theCookie)
	}

	durl := dashboardURL.String()
	t.Log(durl)

	// update headers
	ts.Client.ClearHeaders()
	ts.Client.SetHeaders(sessionHeader)

	html, resp := ts.Client.GetHTML(durl)
	if !a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for dashboard") {
		t.Log(html.Find("body").Text())
	}

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
	})

	testRef := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{0}, 16)}
	ts.RoomState.AddEndpoint(testRef, nil)

	html, resp = ts.Client.GetHTML(durl)
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

	html, resp = ts.Client.GetHTML(durl)
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

func assertCSRFTokenPresent(t *testing.T, sel *goquery.Selection) {
	a := assert.New(t)

	csrfField := sel.Find("input[name=gorilla.csrf.Token]")
	a.EqualValues(1, csrfField.Length(), "no csrf-token input tag")
	tipe, ok := csrfField.Attr("type")
	a.True(ok, "csrf input has a type")
	a.Equal("hidden", tipe, "wrong type on csrf field")
}
