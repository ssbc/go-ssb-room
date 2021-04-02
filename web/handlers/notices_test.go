package handlers

import (
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	a, r := assert.New(t), require.New(t)

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
	// cookiejar: a very cheap client session
	// TODO: refactor login dance for re-use in testing / across tests
	jar, err := cookiejar.New(nil)
	r.NoError(err)

	// when dealing with cookies we also need to have an Host and URL-Scheme
	// for the jar to save and load them correctly
	formEndpoint := ts.URLTo(router.AuthFallbackLogin)

	doc, resp = ts.Client.GetHTML(formEndpoint)
	a.Equal(http.StatusOK, resp.Code)

	csrfCookie := resp.Result().Cookies()
	a.Len(csrfCookie, 1, "should have one cookie for CSRF protection validation")
	t.Log(csrfCookie)
	jar.SetCookies(formEndpoint, csrfCookie)

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

	// have the database return okay for any user
	testUser := roomdb.Member{ID: 23}
	ts.AuthFallbackDB.CheckReturns(testUser.ID, nil)
	ts.MembersDB.GetByIDReturns(testUser, nil)

	postEndpoint := ts.URLTo(router.AuthFallbackFinalize)

	// construct HTTP Header with Referer and Cookie
	var csrfCookieHeader = http.Header(map[string][]string{})
	csrfCookieHeader.Set("Referer", "https://localhost")
	cs := jar.Cookies(postEndpoint)
	r.Len(cs, 1, "expecting one cookie for csrf")
	theCookie := cs[0].String()
	a.NotEqual("", theCookie, "should have a new cookie")
	csrfCookieHeader.Set("Cookie", theCookie)
	ts.Client.SetHeaders(csrfCookieHeader)

	resp = ts.Client.PostForm(postEndpoint, loginVals)
	a.Equal(http.StatusSeeOther, resp.Code, "wrong HTTP status code for sign in")

	sessionCookie := resp.Result().Cookies()
	jar.SetCookies(postEndpoint, sessionCookie)

	var sessionHeader = http.Header(map[string][]string{})
	cs = jar.Cookies(noticeURL)
	// TODO: why doesnt this return the csrf cookie?!
	r.NotEqual(len(cs), 0, "expecting a cookie!")
	for _, c := range cs {
		theCookie := c.String()
		a.NotEqual("", theCookie, "should have a new cookie")
		sessionHeader.Add("Cookie", theCookie)
	}

	// update headers
	ts.Client.ClearHeaders()
	ts.Client.SetHeaders(sessionHeader)

	cnt := ts.MembersDB.GetByIDCallCount()
	// now we are logged in, anchor tag should be there
	doc, resp = ts.Client.GetHTML(noticeURL)
	a.Equal(http.StatusOK, resp.Code)
	a.Equal(cnt+1, ts.MembersDB.GetByIDCallCount())

	a.EqualValues(1, doc.Find(editButtonSelector).Length())
}
