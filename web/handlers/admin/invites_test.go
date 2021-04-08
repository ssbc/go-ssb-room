package admin

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
)

func TestInvitesOverview(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testUser := roomdb.Member{ID: 23}

	lst := []roomdb.Invite{
		{ID: 1, CreatedBy: testUser},
		{ID: 2, CreatedBy: testUser},
		{ID: 3, CreatedBy: testUser},
	}
	ts.InvitesDB.ListReturns(lst, nil)

	html, resp := ts.Client.GetHTML("/invites")
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminInvitesWelcome"},
		{"title", "AdminInvitesTitle"},
		{"#invite-list-count", "AdminInvitesCountPlural"},
	})

	// devided by two because there is one for wide and one for slim/mobile
	trSelector := "#the-table-rows tr"
	a.EqualValues(3, html.Find(trSelector).Length()/2, "wrong number of entries on the table (plural)")

	lst = []roomdb.Invite{
		{ID: 666, CreatedBy: testUser},
	}
	ts.InvitesDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML("/invites")
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminInvitesWelcome"},
		{"title", "AdminInvitesTitle"},
		{"#invite-list-count", "AdminInvitesCountSingular"},
	})

	elems := html.Find(trSelector)
	a.EqualValues(1, elems.Length()/2, "wrong number of entries on the table (signular)")

	// check for link to remove confirm link
	link, yes := elems.Find("a").Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/invites/revoke/confirm?id=666", link)
}

func TestInvitesCreateForm(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminInvitesOverview).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
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
}

func TestInvitesCreate(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	r := require.New(t)

	urlTo := web.NewURLTo(ts.Router)
	urlRemove := urlTo(router.AdminInvitesCreate)

	testInvite := "your-fake-test-invite"
	ts.InvitesDB.CreateReturns(testInvite, nil)

	rec := ts.Client.PostForm(urlRemove.String(), url.Values{})
	a.Equal(http.StatusOK, rec.Code)

	r.Equal(1, ts.InvitesDB.CreateCallCount(), "expected one invites.Create call")
	_, userID := ts.InvitesDB.CreateArgsForCall(0)
	a.EqualValues(ts.User.ID, userID)

	doc, err := goquery.NewDocumentFromReader(rec.Body)
	require.NoError(t, err, "failed to parse response")

	webassert.Localized(t, doc, []webassert.LocalizedElement{
		{"title", "AdminInviteCreatedTitle"},
		{"#welcome", "AdminInviteCreatedTitle" + "AdminInviteCreatedInstruct"},
	})

	wantURL := urlTo(router.CompleteInviteFacade, "token", testInvite)
	wantURL.Host = ts.Domain
	wantURL.Scheme = "https"

	shownLink := doc.Find("#invite-facade-link").Text()
	a.Equal(wantURL.String(), shownLink)
}
