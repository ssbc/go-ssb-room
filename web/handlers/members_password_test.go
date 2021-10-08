// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/webassert"
)

func TestLoginAndChangePassword(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	signInFormURL := ts.URLTo(router.AuthFallbackLogin)

	doc, resp := ts.Client.GetHTML(signInFormURL)
	a.Equal(http.StatusOK, resp.Code)

	csrfCookie := resp.Result().Cookies()
	a.True(len(csrfCookie) > 0, "should have one cookie for CSRF protection validation")

	passwordForm := doc.Find("#password-fallback")
	webassert.CSRFTokenPresent(t, passwordForm)

	csrfTokenElem := passwordForm.Find("input[type=hidden]")
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
	ts.MembersDB.GetByIDReturns(roomdb.Member{ID: 23}, nil)

	signInURL := ts.URLTo(router.AuthFallbackFinalize)

	// important for CSRF
	var refererHeader = make(http.Header)
	refererHeader.Set("Referer", "https://localhost")
	ts.Client.SetHeaders(refererHeader)

	resp = ts.Client.PostForm(signInURL, loginVals)
	a.Equal(http.StatusSeeOther, resp.Code, "wrong HTTP status code for sign in")

	a.Equal(1, ts.AuthFallbackDB.CheckCallCount())

	// now request the protected dashboard page
	dashboardURL := ts.URLTo(router.AdminDashboard)

	html, resp := ts.Client.GetHTML(dashboardURL)
	if !a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for dashboard") {
		t.Log(html.Find("body").Text())
	}

	// check the link to the own details is there
	gotDetailsPageURL, has := html.Find("#own-details-page").Attr("href")
	a.True(has, "did not get href for own details page")

	wantDetailsPageURL := ts.URLTo(router.AdminMemberDetails, "id", "23")
	a.Equal(wantDetailsPageURL.String(), gotDetailsPageURL)

	// check the details page has the link to change the password
	html, resp = ts.Client.GetHTML(wantDetailsPageURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for own details page")

	gotChangePasswordURL, has := html.Find("#change-password").Attr("href")
	a.True(has, "did not get href for pw change page")

	wantChangePasswordURL := ts.URLTo(router.MembersChangePasswordForm)
	a.Equal(wantChangePasswordURL.String(), gotChangePasswordURL)

	// query the form to assert the form and get a csrf token
	html, resp = ts.Client.GetHTML(wantChangePasswordURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for change password form")

	pwForm := html.Find("#change-password")

	postData := webassert.CSRFTokenPresent(t, pwForm)

	webassert.ElementsInForm(t, pwForm, []webassert.FormElement{
		{Name: "new-password", Type: "password"},
		{Name: "repeat-password", Type: "password"},
	})

	// construct the password change request(s)

	testPassword := "our-super-secret-new-password"

	// first we make sure they need to match
	postData.Set("new-password", testPassword)
	postData.Set("repeat-password", testPassword+"-whoops")
	resp = ts.Client.PostForm(wantChangePasswordURL, postData)
	a.Equal(http.StatusSeeOther, resp.Code) // redirects back with a flash message
	webassert.HasFlashMessages(t, ts.Client, wantChangePasswordURL, "ErrorPasswordDidntMatch")
	a.Equal(0, ts.AuthFallbackDB.SetPasswordCallCount(), "shouldnt call database")

	// now check it can't be too short
	postData.Set("new-password", "nope")
	postData.Set("repeat-password", "nope")
	resp = ts.Client.PostForm(wantChangePasswordURL, postData)
	a.Equal(http.StatusSeeOther, resp.Code)
	webassert.HasFlashMessages(t, ts.Client, wantChangePasswordURL, "ErrorPasswordTooShort")
	a.Equal(0, ts.AuthFallbackDB.SetPasswordCallCount(), "shouldnt call database")

	// now check it goes through
	postData.Set("new-password", testPassword)
	postData.Set("repeat-password", testPassword)
	resp = ts.Client.PostForm(wantChangePasswordURL, postData)
	a.Equal(http.StatusSeeOther, resp.Code)
	webassert.HasFlashMessages(t, ts.Client, wantChangePasswordURL, "AuthFallbackPasswordUpdated")
	a.Equal(1, ts.AuthFallbackDB.SetPasswordCallCount(), "should have called the database")
	_, mid, pw := ts.AuthFallbackDB.SetPasswordArgsForCall(0)
	a.EqualValues(23, mid)
	a.EqualValues(testPassword, pw)
}

func TestChangePasswordWithToken(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	testToken := "foo-bar"
	changePasswordURL := ts.URLTo(router.MembersChangePasswordForm, "token", testToken)

	// query the form to assert the form and get a csrf token
	html, resp := ts.Client.GetHTML(changePasswordURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for change password form")

	pwForm := html.Find("#change-password")

	// important for CSRF
	var refererHeader = make(http.Header)
	refererHeader.Set("Referer", "https://localhost")
	ts.Client.SetHeaders(refererHeader)
	postData := webassert.CSRFTokenPresent(t, pwForm)

	webassert.ElementsInForm(t, pwForm, []webassert.FormElement{
		{Name: "new-password", Type: "password"},
		{Name: "repeat-password", Type: "password"},
		{Name: "reset-token", Type: "hidden", Value: testToken},
	})

	// construct the password change request(s)

	postData.Set("reset-token", testToken)

	testPassword := "our-super-secret-new-password"

	// first we make sure they need to match
	postData.Set("new-password", testPassword)
	postData.Set("repeat-password", testPassword+"-whoops")
	resp = ts.Client.PostForm(changePasswordURL, postData)
	a.Equal(http.StatusSeeOther, resp.Code) // redirects back with a flash message
	webassert.HasFlashMessages(t, ts.Client, changePasswordURL, "ErrorPasswordDidntMatch")
	a.Equal(0, ts.AuthFallbackDB.SetPasswordWithTokenCallCount(), "shouldnt call database")

	// now check it can't be too short
	postData.Set("new-password", "nope")
	postData.Set("repeat-password", "nope")
	resp = ts.Client.PostForm(changePasswordURL, postData)
	a.Equal(http.StatusSeeOther, resp.Code)
	webassert.HasFlashMessages(t, ts.Client, changePasswordURL, "ErrorPasswordTooShort")
	a.Equal(0, ts.AuthFallbackDB.SetPasswordWithTokenCallCount(), "shouldnt call database")

	// now check it goes through
	postData.Set("new-password", testPassword)
	postData.Set("repeat-password", testPassword)
	resp = ts.Client.PostForm(changePasswordURL, postData)
	a.Equal(http.StatusSeeOther, resp.Code)
	webassert.HasFlashMessages(t, ts.Client, changePasswordURL, "AuthFallbackPasswordUpdated")
	a.Equal(1, ts.AuthFallbackDB.SetPasswordWithTokenCallCount(), "should have called the database")
	_, gotTok, gotPassword := ts.AuthFallbackDB.SetPasswordWithTokenArgsForCall(0)
	a.EqualValues(testPassword, gotPassword)
	a.EqualValues(testToken, gotTok)
}
