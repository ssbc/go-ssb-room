package handlers

import (
	"bytes"
	"context"
	"encoding/base64"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.cryptoscope.co/muxrpc/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/internal/maybemod/keys"
	"github.com/ssb-ngi-pointer/go-ssb-room/internal/signinwithssb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	refs "go.mindeco.de/ssb-refs"
)

func TestRestricted(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	testURLs := []string{
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

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AuthFallbackWelcome"},
		{"title", "AuthFallbackTitle"},
	})
}

func TestFallbackAuth(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	// very cheap "browser" client session
	jar, err := cookiejar.New(nil)
	r.NoError(err)

	signInFormURL, err := ts.Router.Get(router.AuthFallbackSignInForm).URL()
	r.Nil(err)
	signInFormURL.Host = "localhost"
	signInFormURL.Scheme = "https"

	doc, resp := ts.Client.GetHTML(signInFormURL.String())
	a.Equal(http.StatusOK, resp.Code)

	csrfCookie := resp.Result().Cookies()
	a.Len(csrfCookie, 1, "should have one cookie for CSRF protection validation")

	jar.SetCookies(signInFormURL, csrfCookie)

	webassert.CSRFTokenPresent(t, doc.Find("form"))

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

	// now request the protected dashboard page
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

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
	})

	testRef := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{0}, 16)}
	ts.RoomState.AddEndpoint(testRef, nil)

	html, resp = ts.Client.GetHTML(durl)
	if !a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code") {
		t.Log(html.Find("body").Text())
	}
	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
		{"#roomCount", "AdminRoomCountSingular"},
	})

	testRef2 := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{1}, 16)}
	ts.RoomState.AddEndpoint(testRef2, nil)

	html, resp = ts.Client.GetHTML(durl)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
		{"#roomCount", "AdminRoomCountPlural"},
	})
}

func TestAuthWithSSBNotConnected(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	// the client is a member but not connected right now
	ts.MembersDB.GetByFeedReturns(roomdb.Member{ID: 1234, Nickname: "test-member"}, nil)
	ts.MockedEndpoints.GetEndpointForReturns(nil, false)

	client, err := keys.NewKeyPair(nil)
	r.NoError(err)

	cc := signinwithssb.GenerateChallenge()

	urlTo := web.NewURLTo(ts.Router)

	signInStartURL := urlTo(router.AuthWithSSBSignIn,
		"cid", client.Feed.Ref(),
		"challenge", cc,
	)
	r.NotNil(signInStartURL)

	t.Log(signInStartURL.String())
	doc, resp := ts.Client.GetHTML(signInStartURL.String())
	a.Equal(http.StatusInternalServerError, resp.Code) // TODO: StatusForbidden

	webassert.Localized(t, doc, []webassert.LocalizedElement{
		// {"#welcome", "AuthWithSSBWelcome"},
		// {"title", "AuthWithSSBTitle"},
	})
}

func TestAuthWithSSBNotAllowed(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	// the client isnt a member
	ts.MembersDB.GetByFeedReturns(roomdb.Member{}, roomdb.ErrNotFound)
	ts.MockedEndpoints.GetEndpointForReturns(nil, false)

	client, err := keys.NewKeyPair(nil)
	r.NoError(err)

	cc := signinwithssb.GenerateChallenge()

	urlTo := web.NewURLTo(ts.Router)

	signInStartURL := urlTo(router.AuthWithSSBSignIn,
		"cid", client.Feed.Ref(),
		"challenge", cc,
	)
	r.NotNil(signInStartURL)

	t.Log(signInStartURL.String())
	doc, resp := ts.Client.GetHTML(signInStartURL.String())
	a.Equal(http.StatusInternalServerError, resp.Code) // TODO: StatusForbidden

	webassert.Localized(t, doc, []webassert.LocalizedElement{
		// {"#welcome", "AuthWithSSBWelcome"},
		// {"title", "AuthWithSSBTitle"},
	})
}

func TestAuthWithSSBHasClient(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	// very cheap "browser" client session
	jar, err := cookiejar.New(nil)
	r.NoError(err)

	// the request to be signed later
	var req signinwithssb.ClientRequest
	req.ServerID = ts.NetworkInfo.RoomID

	// the keypair for our client
	testMember := roomdb.Member{ID: 1234, Nickname: "test-member"}
	client, err := keys.NewKeyPair(nil)
	r.NoError(err)
	testMember.PubKey = client.Feed

	// setup the mocked database
	ts.MembersDB.GetByFeedReturns(testMember, nil)
	ts.AuthWithSSB.CreateTokenReturns("abcdefgh", nil)
	ts.AuthWithSSB.CheckTokenReturns(testMember.ID, nil)
	ts.MembersDB.GetByIDReturns(testMember, nil)

	// fill the basic infos of the request
	req.ClientID = client.Feed

	// this is our fake "connected" client
	var edp muxrpc.FakeEndpoint

	// setup a mocked muxrpc call that asserts the arguments and returns the needed signature
	edp.AsyncCalls(func(_ context.Context, ret interface{}, encoding muxrpc.RequestEncoding, method muxrpc.Method, args ...interface{}) error {
		a.Equal(muxrpc.TypeString, encoding)
		a.Equal("httpAuth.requestSolution", method.String())

		r.Len(args, 2, "expected two args")

		serverChallenge, ok := args[0].(string)
		r.True(ok, "argument[0] is not a string: %T", args[0])
		a.NotEqual("", serverChallenge)
		// update the challenge
		req.ServerChallenge = serverChallenge

		clientChallenge, ok := args[1].(string)
		r.True(ok, "argument[1] is not a string: %T", args[1])
		a.Equal(req.ClientChallenge, clientChallenge)

		strptr, ok := ret.(*string)
		r.True(ok, "return is not a string pointer: %T", ret)

		// sign the request now that we have the sc
		clientSig := req.Sign(client.Pair.Secret)

		*strptr = base64.URLEncoding.EncodeToString(clientSig)
		return nil
	})

	// setup the fake client endpoint
	ts.MockedEndpoints.GetEndpointForReturns(&edp, true)

	cc := signinwithssb.GenerateChallenge()
	// update the challenge
	req.ClientChallenge = cc

	// prepare the url
	signInStartURL := web.NewURLTo(ts.Router)(router.AuthWithSSBSignIn,
		"cid", client.Feed.Ref(),
		"cc", cc,
	)
	signInStartURL.Host = "localhost"
	signInStartURL.Scheme = "https"

	r.NotNil(signInStartURL)

	t.Log(signInStartURL.String())
	doc, resp := ts.Client.GetHTML(signInStartURL.String())
	a.Equal(http.StatusOK, resp.Code)

	webassert.Localized(t, doc, []webassert.LocalizedElement{
		// {"#welcome", "AuthWithSSBWelcome"},
		// {"title", "AuthWithSSBTitle"},
	})

	// analyse the endpoints call
	a.Equal(1, ts.MockedEndpoints.GetEndpointForCallCount())
	edpRef := ts.MockedEndpoints.GetEndpointForArgsForCall(0)
	a.Equal(client.Feed.Ref(), edpRef.Ref())

	// check the mock was called
	a.Equal(1, edp.AsyncCallCount())

	// check that we have a new cookie
	sessionCookie := resp.Result().Cookies()
	r.True(len(sessionCookie) > 0, "expecting one cookie!")
	jar.SetCookies(signInStartURL, sessionCookie)

	// now request the protected dashboard page
	dashboardURL, err := ts.Router.Get(router.AdminDashboard).URL()
	r.Nil(err)
	dashboardURL.Host = "localhost"
	dashboardURL.Scheme = "https"

	var sessionHeader = http.Header(map[string][]string{})
	cs := jar.Cookies(dashboardURL)

	r.True(len(cs) > 0, "expecting one cookie!")
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

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDashboardWelcome"},
		{"title", "AdminDashboardTitle"},
	})

}
