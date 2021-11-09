// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/webassert"
	refs "go.mindeco.de/ssb-refs"
)

func TestInviteShowAcceptForm(t *testing.T) {
	ts := setup(t)

	t.Run("token doesnt exist", func(t *testing.T) {
		a, r := assert.New(t), require.New(t)

		testToken := "nonexistant-test-token"
		acceptURL404 := ts.URLTo(router.CompleteInviteFacade, "token", testToken)
		r.NotNil(acceptURL404)

		// prep the mocked db for http:404
		ts.InvitesDB.GetByTokenReturns(roomdb.Invite{}, roomdb.ErrNotFound)

		// request the form
		doc, resp := ts.Client.GetHTML(acceptURL404)
		// 500 until https://github.com/ssb-ngi-pointer/go-ssb-room/issues/66 is fixed
		a.Equal(http.StatusInternalServerError, resp.Code)

		// check database calls
		r.EqualValues(1, ts.InvitesDB.GetByTokenCallCount())
		_, tokenFromArg := ts.InvitesDB.GetByTokenArgsForCall(0)
		a.Equal(testToken, tokenFromArg)

		// fix #66
		// assertLocalized(t, doc, []localizedElement{
		// 	{"#welcome", "AuthFallbackWelcome"},
		// 	{"title", "AuthFallbackTitle"},
		// })
		gotErr := doc.Find("#errBody").Text()
		wantErr := weberrors.ErrNotFound{What: "invite"}
		a.EqualError(wantErr, gotErr)
	})

	t.Run("token DOES exist", func(t *testing.T) {
		a, r := assert.New(t), require.New(t)

		testToken := "existing-test-token"
		validAcceptURL := ts.URLTo(router.CompleteInviteFacade, "token", testToken)

		// prep the mocked db for http:200
		fakeExistingInvite := roomdb.Invite{ID: 1234}
		ts.InvitesDB.GetByTokenReturns(fakeExistingInvite, nil)

		// request the form
		doc, resp := ts.Client.GetHTML(validAcceptURL)
		a.Equal(http.StatusOK, resp.Code)

		// check database calls
		r.EqualValues(2, ts.InvitesDB.GetByTokenCallCount())
		_, tokenFromArg := ts.InvitesDB.GetByTokenArgsForCall(1)
		a.Equal(testToken, tokenFromArg)

		webassert.Localized(t, doc, []webassert.LocalizedElement{
			{"#claim-invite-uri", "InviteFacadeJoin"},
			{"title", "InviteFacadeTitle"},
		})

		// Fallback URL in data-href-fallback
		fallbackURL := ts.URLTo(router.CompleteInviteFacadeFallback, "token", testToken)
		joinDataHrefFallback, ok := doc.Find("#claim-invite-uri").Attr("data-href-fallback")
		a.Equal(fallbackURL.String(), joinDataHrefFallback)
		a.True(ok)

		// ssb-uri in href
		joinDataHref, ok := doc.Find("#claim-invite-uri").Attr("href")
		a.True(ok)
		joinURI, err := url.Parse(joinDataHref)
		r.NoError(err)

		a.Equal("ssb", joinURI.Scheme)
		a.Equal("experimental", joinURI.Opaque)

		params := joinURI.Query()
		a.Equal("claim-http-invite", params.Get("action"))

		inviteParam := params.Get("invite")
		a.Equal(testToken, inviteParam)

		postTo := params.Get("postTo")
		expectedConsumeInviteURL := ts.URLTo(router.CompleteInviteConsume)
		a.Equal(expectedConsumeInviteURL.String(), postTo)
	})
}

func TestInviteShowAcceptFormOnAndroid(t *testing.T) {
	ts := setup(t)

	a, r := assert.New(t), require.New(t)

	testToken := "existing-test-token"
	validAcceptURL := ts.URLTo(router.CompleteInviteFacade, "token", testToken)

	// Mimic Android Chrome
	var uaHeader = make(http.Header)
	uaHeader.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5 Build/MOB30H) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.133 Mobile Safari/537.36")
	ts.Client.SetHeaders(uaHeader)

	// prep the mocked db for http:200
	fakeExistingInvite := roomdb.Invite{ID: 1234}
	ts.InvitesDB.GetByTokenReturns(fakeExistingInvite, nil)

	// request the form
	doc, resp := ts.Client.GetHTML(validAcceptURL)
	a.Equal(http.StatusOK, resp.Code)

	// check database calls
	r.EqualValues(1, ts.InvitesDB.GetByTokenCallCount())
	_, tokenFromArg := ts.InvitesDB.GetByTokenArgsForCall(0)
	a.Equal(testToken, tokenFromArg)

	webassert.Localized(t, doc, []webassert.LocalizedElement{
		{"#claim-invite-uri", "InviteFacadeJoin"},
		{"title", "InviteFacadeTitle"},
	})

	// ssb-uri in href
	joinDataHref, ok := doc.Find("#claim-invite-uri").Attr("href")
	a.True(ok)
	joinURI, err := url.Parse(joinDataHref)
	r.NoError(err)

	a.Equal("intent", joinURI.Scheme)
	a.Equal("experimental", joinURI.Host)

	params := joinURI.Query()
	a.Equal("claim-http-invite", params.Get("action"))

	inviteParam := params.Get("invite")
	a.Equal(testToken, inviteParam)

	postTo := params.Get("postTo")
	expectedConsumeInviteURL := ts.URLTo(router.CompleteInviteConsume)
	a.Equal(expectedConsumeInviteURL.String(), postTo)

	frag := joinURI.Fragment
	a.Equal("Intent;scheme=ssb;package=se.manyver;end;", frag)
}

func TestInviteConsumeInviteHTTP(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	testToken := "existing-test-token-2"
	validAcceptURL := ts.URLTo(router.CompleteInviteInsertID, "token", testToken)
	testInvite := roomdb.Invite{ID: 4321}
	ts.InvitesDB.GetByTokenReturns(testInvite, nil)

	// request the form (for a valid csrf token)
	doc, resp := ts.Client.GetHTML(validAcceptURL)
	a.Equal(http.StatusOK, resp.Code)

	form := doc.Find("form#inviteConsume")
	r.Equal(1, form.Length())

	consumeInviteURLString, has := form.Attr("action")
	a.True(has, "form should have an action attribute")
	expectedConsumeInviteURL := ts.URLTo(router.CompleteInviteConsume)
	a.Equal(expectedConsumeInviteURL.String(), consumeInviteURLString)

	webassert.CSRFTokenPresent(t, form)
	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "invite", Type: "hidden", Value: testToken},
		{Name: "id", Type: "text"},
	})

	// get the corresponding token from the page
	csrfTokenElem := form.Find(`input[name="gorilla.csrf.Token"]`)
	a.Equal(1, csrfTokenElem.Length())
	csrfName, has := csrfTokenElem.Attr("name")
	a.True(has, "should have a name attribute")
	csrfValue, has := csrfTokenElem.Attr("value")
	a.True(has, "should have value attribute")

	// create the consume request
	testNewMember := refs.FeedRef{
		ID:   bytes.Repeat([]byte{1}, 32),
		Algo: refs.RefAlgoFeedSSB1,
	}
	consumeVals := url.Values{
		"invite": []string{testToken},
		"id":     []string{testNewMember.Ref()},

		csrfName: []string{csrfValue},
	}

	// construct the consume endpoint url
	consumeInviteURL := ts.URLTo(router.CompleteInviteConsume)

	// construct the header with the Referer or csrf check
	var csrfCookieHeader = http.Header(map[string][]string{})
	csrfCookieHeader.Set("Referer", "https://localhost")
	ts.Client.SetHeaders(csrfCookieHeader)

	// prepare the mock
	ts.InvitesDB.ConsumeReturns(testInvite, nil)

	// send the POST
	resp = ts.Client.PostForm(consumeInviteURL, consumeVals)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for sign in")

	// check how consume was called
	r.EqualValues(1, ts.InvitesDB.ConsumeCallCount())
	_, tokenFromArg, newMemberRef := ts.InvitesDB.ConsumeArgsForCall(0)
	a.Equal(testToken, tokenFromArg)
	a.True(newMemberRef.Equal(&testNewMember))
}

func TestInviteConsumeInviteJSON(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	testToken := "existing-test-token-2"

	testInvite := roomdb.Invite{ID: 4321}
	ts.InvitesDB.GetByTokenReturns(testInvite, nil)

	// check if the token is still valid
	checkInviteURL := ts.URLTo(router.CompleteInviteFacade)
	qvals := url.Values{
		"token":    []string{testToken},
		"encoding": []string{"json"},
	}
	checkInviteURL.RawQuery = qvals.Encode()

	// send the request and check the json
	resp := ts.Client.GetBody(checkInviteURL)
	result := resp.Result()
	a.Equal(http.StatusOK, result.StatusCode)

	var reply struct {
		Invite string
		PostTo string
	}
	err := json.NewDecoder(result.Body).Decode(&reply)
	r.NoError(err)

	// construct the consume endpoint url
	consumeInviteURL := ts.URLTo(router.CompleteInviteConsume)

	a.Equal(consumeInviteURL.String(), reply.PostTo, "wrong postTo in JSON body")
	a.Equal(testToken, reply.Invite, "wrong invite token")

	// create the consume request
	testNewMember := refs.FeedRef{
		ID:   bytes.Repeat([]byte{1}, 32),
		Algo: refs.RefAlgoFeedSSB1,
	}
	var consume inviteConsumePayload
	consume.Invite = testToken
	consume.ID = testNewMember

	// prepare the mock
	ts.InvitesDB.ConsumeReturns(testInvite, nil)

	// send the POST
	resp = ts.Client.SendJSON(consumeInviteURL, consume)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code for sign in")

	// check how consume was called
	r.EqualValues(1, ts.InvitesDB.ConsumeCallCount())
	_, tokenFromArg, newMemberRef := ts.InvitesDB.ConsumeArgsForCall(0)
	a.Equal(testToken, tokenFromArg)
	a.True(newMemberRef.Equal(&testNewMember))

	var jsonConsumeResp inviteConsumeJSONResponse
	err = json.NewDecoder(resp.Body).Decode(&jsonConsumeResp)
	r.NoError(err)

	a.Equal("successful", jsonConsumeResp.Status)

	gotRA := jsonConsumeResp.RoomAddress
	a.True(strings.HasPrefix(gotRA, "net:localhost:8008~shs:"), "not for the test host: %s", gotRA)
	a.True(strings.HasSuffix(gotRA, base64.StdEncoding.EncodeToString(ts.NetworkInfo.RoomID.PubKey())), "public key missing? %s", gotRA)
}

func TestInviteConsumptionDenied(t *testing.T) {
	ts := setup(t)
	a, r := assert.New(t), require.New(t)

	testToken := "existing-test-token-2"
	validAcceptURL := ts.URLTo(router.CompleteInviteFacade, "token", testToken)
	r.NotNil(validAcceptURL)

	testInvite := roomdb.Invite{ID: 4321}
	ts.InvitesDB.GetByTokenReturns(testInvite, nil)

	ts.DeniedKeysDB.HasFeedReturns(true)

	// create the consume request
	testNewMember := refs.FeedRef{
		ID:   bytes.Repeat([]byte{1}, 32),
		Algo: refs.RefAlgoFeedSSB1,
	}

	var consume inviteConsumePayload
	consume.Invite = testToken
	consume.ID = testNewMember

	// construct the consume endpoint url
	consumeInviteURL := ts.URLTo(router.CompleteInviteConsume)
	r.NotNil(consumeInviteURL)

	// prepare the mock
	ts.InvitesDB.ConsumeReturns(testInvite, nil)

	// send the POST
	resp := ts.Client.SendJSON(consumeInviteURL, consume)

	// decode the json response
	var jsonConsumeResp inviteConsumeJSONResponse
	err := json.NewDecoder(resp.Body).Decode(&jsonConsumeResp)
	r.NoError(err)

	// json response should indicate an error for the denied key
	a.Equal("error", jsonConsumeResp.Status)

	// invite should not be consumed
	r.EqualValues(0, ts.InvitesDB.ConsumeCallCount())
}
