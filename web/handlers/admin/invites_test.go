// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web/router"
	"github.com/ssbc/go-ssb-room/v2/web/webassert"
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

	invitesOverviewURL := ts.URLTo(router.AdminInvitesOverview)

	html, resp := ts.Client.GetHTML(invitesOverviewURL)
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

	html, resp = ts.Client.GetHTML(invitesOverviewURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminInvitesWelcome"},
		{"title", "AdminInvitesTitle"},
		{"#invite-list-count", "AdminInvitesCountSingular"},
	})

	elems := html.Find(trSelector)
	a.EqualValues(1, elems.Length()/2, "wrong number of entries on the table (signular)")

	// check for the link to member details
	link, yes := elems.Find("a").Eq(0).Attr("href")
	a.True(yes, "a-tag has href attribute")
	wantURL := ts.URLTo(router.AdminMemberDetails, "id", testUser.ID)
	a.Equal(wantURL.String(), link)

	// check for link to remove confirm link
	link, yes = elems.Find("a").Eq(1).Attr("href")
	a.True(yes, "a-tag has href attribute")
	wantURL = ts.URLTo(router.AdminInvitesRevokeConfirm, "id", 666)
	a.Equal(wantURL.String(), link)

	testInviteButtonDisabled := func(shouldBeDisabled bool) {
		html, resp = ts.Client.GetHTML(invitesOverviewURL)
		a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
		inviteButton := html.Find("#create-invite button")
		_, disabled := inviteButton.Attr("disabled")
		a.EqualValues(shouldBeDisabled, disabled, "invite button should be disabled")
	}

	// member, mod, admin should all be able to invite in ModeCommunity
	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeCommunity, nil)
	ts.User = roomdb.Member{
		ID:   1234,
		Role: roomdb.RoleAdmin,
	}
	testInviteButtonDisabled(false)

	ts.User = roomdb.Member{
		ID:   7331,
		Role: roomdb.RoleModerator,
	}
	testInviteButtonDisabled(false)

	ts.User = roomdb.Member{
		ID:   9001,
		Role: roomdb.RoleMember,
	}
	testInviteButtonDisabled(false)

	// mod and admin should be able to invite, member should not
	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeRestricted, nil)
	ts.User = roomdb.Member{
		ID:   1234,
		Role: roomdb.RoleAdmin,
	}
	testInviteButtonDisabled(false)

	ts.User = roomdb.Member{
		ID:   7331,
		Role: roomdb.RoleModerator,
	}
	testInviteButtonDisabled(false)

	ts.User = roomdb.Member{
		ID:   9001,
		Role: roomdb.RoleMember,
	}
	testInviteButtonDisabled(true)
}

func TestInvitesCreateForm(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	overviewURL := ts.URLTo(router.AdminInvitesOverview)

	html, resp := ts.Client.GetHTML(overviewURL)
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

	addURL := ts.URLTo(router.AdminInvitesCreate)
	a.Equal(addURL.String(), action)
}

func TestInvitesCreateAndRevoke(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	r := require.New(t)

	urlCreate := ts.URLTo(router.AdminInvitesCreate)

	testInvite := "your-fake-test-invite"
	ts.InvitesDB.CreateReturns(testInvite, nil)

	totalCreateCallCount := 0
	createInviteShouldWork := func(t *testing.T, works bool) *httptest.ResponseRecorder {
		a := assert.New(t)
		r := require.New(t)

		rec := ts.Client.PostForm(urlCreate, url.Values{})
		if works {
			totalCreateCallCount += 1
			a.Equal(http.StatusOK, rec.Code)
			r.Equal(totalCreateCallCount, ts.InvitesDB.CreateCallCount())
			_, userID := ts.InvitesDB.CreateArgsForCall(totalCreateCallCount - 1)
			a.EqualValues(ts.User.ID, userID)
		} else {
			a.Equal(http.StatusForbidden, rec.Code)
			r.Equal(totalCreateCallCount, ts.InvitesDB.CreateCallCount())
		}
		return rec
	}

	totalRevokeCallCount := 0
	urlRevoke := ts.URLTo(router.AdminInvitesRevoke)
	canRevokeInvite := func(t *testing.T, canRevoke bool) {
		a := assert.New(t)
		r := require.New(t)

		rec := ts.Client.PostForm(urlRevoke, url.Values{
			"id": []string{"666"},
		})
		a.Equal(http.StatusSeeOther, rec.Code)
		if canRevoke {
			totalRevokeCallCount += 1
			r.Equal(totalRevokeCallCount, ts.InvitesDB.RevokeCallCount())
			_, userID := ts.InvitesDB.RevokeArgsForCall(totalRevokeCallCount - 1)
			a.EqualValues(666, userID)
		} else {
			r.Equal(totalRevokeCallCount, ts.InvitesDB.RevokeCallCount())
		}
		return
	}

	rec := createInviteShouldWork(t, true)

	doc, err := goquery.NewDocumentFromReader(rec.Body)
	r.NoError(err, "failed to parse response")

	webassert.Localized(t, doc, []webassert.LocalizedElement{
		{"title", "AdminInviteCreatedTitle"},
		{"#welcome", "AdminInviteCreatedTitle" + "AdminInviteCreatedInstruct"},
	})

	wantURL := ts.URLTo(router.CompleteInviteFacade, "token", testInvite)

	shownLink := doc.Find("#invite-facade-link").Text()
	a.Equal(wantURL.String(), shownLink)

	memberUser := roomdb.Member{
		ID:     7331,
		Role:   roomdb.RoleMember,
		PubKey: generatePubKey(),
	}
	modUser := roomdb.Member{
		ID:     9001,
		Role:   roomdb.RoleModerator,
		PubKey: generatePubKey(),
	}
	adminUser := roomdb.Member{
		ID:     1337,
		Role:   roomdb.RoleAdmin,
		PubKey: generatePubKey(),
	}

	/* test invite creation under various restricted mode with the roles member, mod, admin */
	for _, mode := range roomdb.AllPrivacyModes {
		t.Run(mode.String(), func(t *testing.T) {
			ts.ConfigDB.GetPrivacyModeReturns(mode, nil)

			// members can only invite in community rooms
			t.Run("members", func(t *testing.T) {
				ts.User = memberUser
				createInviteShouldWork(t, mode != roomdb.ModeRestricted)
				canRevokeInvite(t, mode != roomdb.ModeRestricted)
			})

			// mods & admins can always invite
			t.Run("mods", func(t *testing.T) {
				ts.User = modUser
				createInviteShouldWork(t, true)
				canRevokeInvite(t, true)
			})

			t.Run("admins", func(t *testing.T) {
				ts.User = adminUser
				createInviteShouldWork(t, true)
				canRevokeInvite(t, true)
			})
		})
	}
}
