// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web/router"
	"github.com/ssbc/go-ssb-room/v2/web/webassert"
	"github.com/stretchr/testify/assert"
)

// TestNoticeSmokeTest ensures the most basic notice serving is working
func TestNoticeSmokeTest(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	noticeData := roomdb.Notice{
		ID:    1,
		Title: "Welcome!",
	}

	ts.NoticeDB.GetByIDReturns(noticeData, nil)

	noticeURL := ts.URLTo(router.CompleteNoticeShow, "id", "1")
	html, res := ts.Client.GetHTML(noticeURL)
	a.Equal(http.StatusOK, res.Code, "wrong HTTP status code")
	a.Equal("Welcome!", html.Find("title").Text())
}

func TestNoticeMarkdownServedCorrectly(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	markdown := `
Hello world!

## The loveliest of rooms is here
`
	noticeData := roomdb.Notice{
		ID:      1,
		Title:   "Welcome!",
		Content: markdown,
	}

	ts.NoticeDB.GetByIDReturns(noticeData, nil)

	noticeURL := ts.URLTo(router.CompleteNoticeShow, "id", "1")
	html, res := ts.Client.GetHTML(noticeURL)
	a.Equal(http.StatusOK, res.Code, "wrong HTTP status code")
	a.Equal("Welcome!", html.Find("title").Text())
	a.Equal("The loveliest of rooms is here", html.Find("h2").Text())
}

// First we get the notices page (to see the buttons are NOT there)
// then we log in as an admin and see that the edit links are there.
func TestNoticesEditButtonVisible(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	ts.AliasesDB.ResolveReturns(roomdb.Alias{}, roomdb.ErrNotFound)

	noticeData := roomdb.Notice{
		ID:      42,
		Title:   "Welcome!",
		Content: `super simple conent`,
	}
	ts.NoticeDB.GetByIDReturns(noticeData, nil)

	// first, we confirm that the button is missing when not logged in
	noticeURL := ts.URLTo(router.CompleteNoticeShow, "id", 42)

	editButtonSelector := `#edit-notice`

	doc, resp := ts.Client.GetHTML(noticeURL)
	a.Equal(http.StatusOK, resp.Code)

	// empty selection <=> we have no link
	a.EqualValues(0, doc.Find(editButtonSelector).Length())

	// start preparing the ~login dance~
	// TODO: make this code reusable and share it with the login => /dashboard http:200 test
	// TODO: refactor login dance for re-use in testing / across tests

	// when dealing with cookies we also need to have an Host and URL-Scheme
	// for the jar to save and load them correctly
	formEndpoint := ts.URLTo(router.AuthFallbackLogin)

	doc, resp = ts.Client.GetHTML(formEndpoint)
	a.Equal(http.StatusOK, resp.Code)

	csrfCookie := resp.Result().Cookies()
	a.True(len(csrfCookie) > 0, "should have one cookie for CSRF protection validation")

	csrfTokenElem := doc.Find(`form#password-fallback input[type="hidden"]`)
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

	// have the database return okay for any user
	testUser := roomdb.Member{ID: 23, Role: roomdb.RoleAdmin}
	ts.AuthFallbackDB.CheckReturns(testUser.ID, nil)
	ts.MembersDB.GetByIDReturns(testUser, nil)

	postEndpoint := ts.URLTo(router.AuthFallbackFinalize)

	// construct HTTP Header with Referer and Cookie
	var refererHeader = make(http.Header)
	refererHeader.Set("Referer", "https://localhost")
	ts.Client.SetHeaders(refererHeader)

	resp = ts.Client.PostForm(postEndpoint, loginVals)
	a.Equal(http.StatusSeeOther, resp.Code, "wrong HTTP status code for sign in")

	cnt := ts.MembersDB.GetByIDCallCount()
	// now we are logged in, anchor tag should be there
	doc, resp = ts.Client.GetHTML(noticeURL)
	a.Equal(http.StatusOK, resp.Code)
	a.Equal(cnt+1, ts.MembersDB.GetByIDCallCount())

	a.EqualValues(1, doc.Find(editButtonSelector).Length())
}

func TestNoticesCreateOnlyModsAndHigherInRestricted(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeRestricted, nil)

	// first, we confirm that we can't access the page when not logged in
	draftNotice := ts.URLTo(router.AdminNoticeDraftTranslation, "name", roomdb.NoticeNews)

	doc, resp := ts.Client.GetHTML(draftNotice)
	a.Equal(http.StatusForbidden, resp.Code)

	// start preparing the ~login dance~
	// TODO: make this code reusable and share it with the login => /dashboard http:200 test
	// TODO: refactor login dance for re-use in testing / across tests

	// when dealing with cookies we also need to have an Host and URL-Scheme
	// for the jar to save and load them correctly
	formEndpoint := ts.URLTo(router.AuthFallbackLogin)

	doc, resp = ts.Client.GetHTML(formEndpoint)
	a.Equal(http.StatusOK, resp.Code)

	csrfCookie := resp.Result().Cookies()
	a.True(len(csrfCookie) > 0, "should have one cookie for CSRF protection validation")

	csrfTokenElem := doc.Find(`form#password-fallback input[type="hidden"]`)
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

	// have the database return okay for any user
	testUser := roomdb.Member{ID: 123, Role: roomdb.RoleMember}
	ts.AuthFallbackDB.CheckReturns(testUser.ID, nil)
	ts.MembersDB.GetByIDReturns(testUser, nil)

	postEndpoint := ts.URLTo(router.AuthFallbackFinalize)

	// construct HTTP Header with Referer and Cookie
	var refererHeader = make(http.Header)
	refererHeader.Set("Referer", "https://localhost")
	ts.Client.SetHeaders(refererHeader)

	resp = ts.Client.PostForm(postEndpoint, loginVals)
	a.Equal(http.StatusSeeOther, resp.Code, "wrong HTTP status code for sign in")

	// now we are logged in, but we shouldn't be able to get the draft page
	doc, resp = ts.Client.GetHTML(draftNotice)
	a.Equal(http.StatusSeeOther, resp.Code)

	noticeListURL := ts.URLTo(router.CompleteNoticeList)
	a.Equal(noticeListURL.String(), resp.Header().Get("Location"))
	a.True(len(resp.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, noticeListURL, "ErrorNotAuthorized")

	// also shouldnt be allowed to save/post
	id := []string{"1"}
	title := []string{"SSB Breaking News: This Test Is Great"}
	content := []string{"Absolutely Thrilling Content"}
	language := []string{"en-GB"}

	// POST a correct request to the save handler, and verify that the save was handled using the mock database)
	u := ts.URLTo(router.AdminNoticeSave)
	formValues := url.Values{"id": id, "title": title, "content": content, "language": language, csrfName: []string{csrfValue}}
	resp = ts.Client.PostForm(u, formValues)
	a.Equal(http.StatusSeeOther, resp.Code, "POST should work")
	a.Equal(0, ts.NoticeDB.SaveCallCount(), "noticedb should not save the notice")

	a.Equal(noticeListURL.String(), resp.Header().Get("Location"))
	a.True(len(resp.Result().Cookies()) > 0, "got a cookie")

	webassert.HasFlashMessages(t, ts.Client, noticeListURL, "ErrorNotAuthorized")

}
