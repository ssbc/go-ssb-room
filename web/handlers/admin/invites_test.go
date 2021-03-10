package admin

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
)

func TestInvitesCreateForm(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminInvitesOverview).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminInvitesWelcome"},
		{"title", "AdminInvitesTitle"},
	})

	formSelection := html.Find("form#create-invite")
	a.EqualValues(1, formSelection.Length())

	method, ok := formSelection.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := formSelection.Attr("action")
	a.True(ok, "form has action set")

	addURL, err := ts.Router.Get(router.AdminInvitesCreate).URL()
	a.NoError(err)

	a.Equal(addURL.String(), action)

	inputSelection := formSelection.Find("input[type=text]")
	a.EqualValues(1, inputSelection.Length())

	name, ok := inputSelection.Attr("name")
	a.True(ok, "input has a name")
	a.Equal("alias_suggestion", name, "wrong name on input field")
}

func TestInvitesCreate(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlTo := web.NewURLTo(ts.Router)
	urlRemove := urlTo(router.AdminInvitesCreate)

	testInvite := "your-fake-test-invite"
	ts.InvitesDB.CreateReturns(testInvite, nil)

	rec := ts.Client.PostForm(urlRemove.String(), url.Values{
		"alias_suggestion": []string{"jerry"},
	})
	a.Equal(http.StatusOK, rec.Code)

	a.Equal(1, ts.InvitesDB.CreateCallCount())
	_, userID, aliasSuggestion := ts.InvitesDB.CreateArgsForCall(0)
	a.EqualValues(ts.User.ID, userID)
	a.EqualValues("jerry", aliasSuggestion)

	doc, err := goquery.NewDocumentFromReader(rec.Body)
	require.NoError(t, err, "failed to parse response")

	assertLocalized(t, doc, []localizedElement{
		{"title", "AdminInviteCreatedTitle"},
		{"#welcome", "AdminInviteCreatedWelcome"},
	})

	wantURL := urlTo(router.CompleteInviteAccept, "token", testInvite)
	wantURL.Host = ts.Domain
	wantURL.Scheme = "https"

	shownLink := doc.Find("#invite-accept-link").Text()
	a.Equal(wantURL.String(), shownLink)
}
