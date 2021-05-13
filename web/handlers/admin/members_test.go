package admin

import (
	"bytes"
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	refs "go.mindeco.de/ssb-refs"
)

func TestMembersEmpty(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url := ts.URLTo(router.AdminMembersOverview)

	html, resp := ts.Client.GetHTML(url)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminMembersWelcome"},
		{"title", "AdminMembersTitle"},
		{"#membersCount", "MemberCountPlural"},
	})
}

func TestMembersAdd(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	ts.User = roomdb.Member{
		ID:   1234,
		Role: roomdb.RoleAdmin,
	}

	listURL := ts.URLTo(router.AdminMembersOverview)

	html, resp := ts.Client.GetHTML(listURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	formSelection := html.Find("form#add-entry")
	a.EqualValues(1, formSelection.Length())

	method, ok := formSelection.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := formSelection.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminMembersAdd)
	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, formSelection, []webassert.FormElement{
		{Name: "pub_key", Type: "text"},
	})

	newKey := "@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519"
	addVals := url.Values{
		// just any key that looks valid
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)

	a.Equal(1, ts.MembersDB.AddCallCount())
	_, addedPubKey, addedRole := ts.MembersDB.AddArgsForCall(0)
	a.Equal(newKey, addedPubKey.Ref())
	a.Equal(roomdb.RoleMember, addedRole)

	/* Verify that the inputs are visible/hidden depending on user roles */
	checkInputsAreDisabled := func(shouldBeDisabled bool) {
		html, resp = ts.Client.GetHTML(listURL)
		a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
		inputContainer := html.Find("#add-member-input-container")
		a.Equal(1, inputContainer.Length())
		inputs := inputContainer.Find("input")
		// pubkey
		a.Equal(1, inputs.Length())
		inputs.Each(func(i int, el *goquery.Selection) {
			_, disabled := el.Attr("disabled")
			a.Equal(shouldBeDisabled, disabled)
		})
		button := inputContainer.Find("button")
		a.Equal(1, button.Length())
		button.Each(func(i int, el *goquery.Selection) {
			_, disabled := el.Attr("disabled")
			a.Equal(shouldBeDisabled, disabled)
		})
	}

	// verify that inputs are enabled for RoleAdmin
	checkInputsAreDisabled(false)

	// verify that inputs are enabled for RoleModerator
	ts.User = roomdb.Member{
		ID:   9001,
		Role: roomdb.RoleModerator,
	}
	checkInputsAreDisabled(false)

	// verify that inputs are disabled for RoleMember
	ts.User = roomdb.Member{
		ID:   7331,
		Role: roomdb.RoleMember,
	}
	checkInputsAreDisabled(true)
}

func TestMembersDontAddInvalid(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	addURL := ts.URLTo(router.AdminMembersAdd)

	newKey := "@some-garbage"
	addVals := url.Values{
		"nick":    []string{"some-test-nick"},
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)

	a.Equal(0, ts.MembersDB.AddCallCount())

	listURL := ts.URLTo(router.AdminMembersOverview)
	res := rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	webassert.HasFlashMessages(t, ts.Client, listURL, "ErrorBadRequest")

}

func TestMembers(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	lst := []roomdb.Member{
		{ID: 1, Role: roomdb.RoleMember, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"}},
		{ID: 2, Role: roomdb.RoleModerator, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("1312"), 8), Algo: "test"}},
		{ID: 3, Role: roomdb.RoleAdmin, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte("acab"), 8), Algo: "true"}},
	}
	ts.MembersDB.ListReturns(lst, nil)

	membersOverviewURL := ts.URLTo(router.AdminMembersOverview)

	html, resp := ts.Client.GetHTML(membersOverviewURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminMembersWelcome"},
		{"title", "AdminMembersTitle"},
		{"#membersCount", "MemberCountPlural"},
	})

	elems := html.Find("#theList li")
	a.EqualValues(elems.Length(), 3)

	// check for the roles labels
	a.EqualValues(elems.Find("span[data-role]").Length(), 2)
	a.EqualValues(elems.Find("span[data-role='admin']").Length(), 1)
	a.EqualValues(elems.Find("span[data-role='moderator']").Length(), 1)

	lst = []roomdb.Member{
		{ID: 666, Role: roomdb.RoleAdmin, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{1}, 32), Algo: "one"}},
	}
	ts.MembersDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML(membersOverviewURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminMembersWelcome"},
		{"title", "AdminMembersTitle"},
		{"#membersCount", "MemberCountSingular"},
	})

	elems = html.Find("#theList li")
	a.EqualValues(elems.Length(), 1)

	// check for the admin role label
	roleElem, yes1 := elems.Find("span[data-role]").Eq(0).Attr("data-role")
	a.True(yes1, "elem with data-role exists")
	a.Equal("admin", roleElem)

	// check for link to member details
	link, yes2 := elems.ContentsFiltered("a").Attr("href")
	a.True(yes2, "a-tag has href attribute")
	wantURL := ts.URLTo(router.AdminMemberDetails, "id", 666)
	a.Equal(wantURL.String(), link)
}

func TestMemberDetails(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	feedRef := refs.FeedRef{ID: bytes.Repeat([]byte{0}, 32), Algo: "fake"}
	aliases := []roomdb.Alias{
		{ID: 11, Name: "robert", Feed: feedRef, Signature: bytes.Repeat([]byte{0}, 4)},
		{ID: 21, Name: "bob", Feed: feedRef, Signature: bytes.Repeat([]byte{0}, 4)},
	}

	member := roomdb.Member{
		ID: 1, Role: roomdb.RoleMember, PubKey: feedRef, Aliases: aliases,
	}
	ts.MembersDB.GetByIDReturns(member, nil)

	memberURL := ts.URLTo(router.AdminMemberDetails, "id", "1")

	ts.User = roomdb.Member{
		ID:   1234,
		Role: roomdb.RoleAdmin,
	}

	html, resp := ts.Client.GetHTML(memberURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"title", "AdminMemberDetailsTitle"},
	})

	aliasList := html.Find("#alias-list").Find("a")

	// check for link to resolve 1st Alias
	aliasRobertLink, yes := aliasList.Eq(0).Attr("href")
	a.True(yes, "a-tag has href attribute")

	wantURL := fmt.Sprintf("https://robert.%s", ts.netInfo.Domain)
	a.Equal(wantURL, aliasRobertLink)

	// check for link to revoke 1st Alias
	revokeRobertLink, yes := aliasList.Eq(1).Attr("href")
	a.True(yes, "a-tag has href attribute")
	wantLink := ts.URLTo(router.AdminAliasesRevokeConfirm, "id", 11)
	a.Equal(wantLink.String(), revokeRobertLink)

	// check for link to resolve 1st Alias
	aliasBobLink, yes := aliasList.Eq(2).Attr("href")
	a.True(yes, "a-tag has href attribute")
	wantURL = fmt.Sprintf("https://bob.%s", ts.netInfo.Domain)
	a.Equal(wantURL, aliasBobLink)

	// check for link to revoke 1st Alias
	revokeBobLink, yes := aliasList.Eq(3).Attr("href")
	a.True(yes, "a-tag has href attribute")
	wantLink = ts.URLTo(router.AdminAliasesRevokeConfirm, "id", 21)
	a.Equal(wantLink.String(), revokeBobLink)

	// check for link to Remove member link
	removeLink, yes := html.Find("#remove-member").Attr("href")
	a.True(yes, "a-tag has href attribute")
	wantLink = ts.URLTo(router.AdminMembersRemoveConfirm, "id", 1)
	a.Equal(wantLink.String(), removeLink)

	testChangeRoleDisabledBehaviour := func(t *testing.T, html *goquery.Document, canSee bool) {
		a := assert.New(t)

		// check for SSB ID
		ssbID := html.Find("#ssb-id")
		a.Equal(feedRef.Ref(), ssbID.Text())

		// check for change-role dropdown
		roleDropdown := html.Find("#change-role")
		if canSee {
			a.Equal(1, roleDropdown.Length())
		} else {
			a.Equal(0, roleDropdown.Length())
		}
	}

	testRemoveButtonHiddenBehavior := func(t *testing.T, html *goquery.Document, canSee bool) {
		a := assert.New(t)

		rmButton := html.Find("a#remove-member")
		if canSee {
			a.Equal(1, rmButton.Length())
		} else {
			a.Equal(0, rmButton.Length())
		}
	}
	overviewURL := ts.URLTo(router.AdminMembersOverview)
	removeURL := ts.URLTo(router.AdminMembersRemove)
	totalRemoveCallCount := 0
	testCanDoRemoveBehavior := func(t *testing.T, html *goquery.Document, canDo bool) {
		a := assert.New(t)

		resp := ts.Client.PostForm(removeURL, url.Values{"id": []string{"1"}})
		a.Equal(http.StatusSeeOther, resp.Code, "unexpected status code")

		var wantLabel string
		if canDo {
			totalRemoveCallCount++
			wantLabel = "AdminMemberRemoved"
		} else {
			wantLabel = "ErrorNotAuthorized"
		}
		a.Equal(totalRemoveCallCount, ts.MembersDB.RemoveIDCallCount())
		webassert.HasFlashMessages(t, ts.Client, overviewURL, wantLabel)
	}

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

	for _, mode := range roomdb.AllPrivacyModes {
		t.Run(mode.String(), func(t *testing.T) {
			ts.ConfigDB.GetPrivacyModeReturns(mode, nil)

			// members can only invite in community rooms
			t.Run("members", func(t *testing.T) {
				ts.User = memberUser
				html, resp := ts.Client.GetHTML(memberURL)
				a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
				testChangeRoleDisabledBehaviour(t, html, false)
				testRemoveButtonHiddenBehavior(t, html, false)
				testCanDoRemoveBehavior(t, html, false)
			})

			// mods & admins can always invite
			t.Run("mods", func(t *testing.T) {
				ts.User = modUser
				html, resp := ts.Client.GetHTML(memberURL)
				a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
				testChangeRoleDisabledBehaviour(t, html, true)
				testRemoveButtonHiddenBehavior(t, html, true)
				testCanDoRemoveBehavior(t, html, true)
			})

			t.Run("admins", func(t *testing.T) {
				ts.User = adminUser
				html, resp := ts.Client.GetHTML(memberURL)
				a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
				testChangeRoleDisabledBehaviour(t, html, true)
				testRemoveButtonHiddenBehavior(t, html, true)
				testCanDoRemoveBehavior(t, html, true)
			})
		})
	}
}

func TestMembersRemoveConfirmation(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testKey, err := refs.ParseFeedRef("@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519")
	a.NoError(err)
	testEntry := roomdb.Member{ID: 666, PubKey: *testKey}
	ts.MembersDB.GetByIDReturns(testEntry, nil)

	urlRemoveConfirm := ts.URLTo(router.AdminMembersRemoveConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRemoveConfirm)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.Ref(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminMembersRemove)
	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "id", Type: "hidden", Value: "666"},
	})
}

func TestMembersRemove(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlRemove := ts.URLTo(router.AdminMembersRemove)

	ts.MembersDB.RemoveIDReturns(nil)

	addVals := url.Values{"id": []string{"666"}}
	rec := ts.Client.PostForm(urlRemove, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)

	a.Equal(1, ts.MembersDB.RemoveIDCallCount())
	_, theID := ts.MembersDB.RemoveIDArgsForCall(0)
	a.EqualValues(666, theID)

	listURL := ts.URLTo(router.AdminMembersOverview)
	// check flash message
	res := rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	webassert.HasFlashMessages(t, ts.Client, listURL, "AdminMemberRemoved")

	// now for unknown ID
	ts.MembersDB.RemoveIDReturns(roomdb.ErrNotFound)
	addVals = url.Values{"id": []string{"667"}}
	rec = ts.Client.PostForm(urlRemove, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)

	// check flash message
	res = rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	webassert.HasFlashMessages(t, ts.Client, listURL, "ErrorNotFound")
}

func TestMembersCreateResetToken(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	// setup mock

	ts.MembersDB.GetByIDReturns(roomdb.Member{
		ID:     2342,
		Role:   roomdb.RoleMember,
		PubKey: refs.FeedRef{ID: make([]byte, 32), Algo: refs.RefAlgoFeedSSB1},
	}, nil)

	urlViewDetails := ts.URLTo(router.AdminMemberDetails, "id", "2342")

	doc, resp := ts.Client.GetHTML(urlViewDetails)
	a.Equal(http.StatusOK, resp.Code)

	form := doc.Find("#create-reset-token")
	a.Equal(1, form.Length(), "form missing from page")

	formMethod, hasMethod := form.Attr("method")
	a.True(hasMethod, "missing method")
	a.Equal(http.MethodPost, formMethod, "wrong method")

	formAction, hasAction := form.Attr("action")
	a.True(hasAction, "missing action")

	resetURL := ts.URLTo(router.AdminMembersCreateFallbackReset)
	a.Equal(resetURL.String(), formAction, "wrong action")

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "member_id", Value: "2342", Type: "hidden"},
	})

	// now create the reset link

	ts.User.Role = roomdb.RoleAdmin

	testToken := "super-secure-token"
	ts.FallbackDB.CreateResetTokenReturns(testToken, nil)

	resp = ts.Client.PostForm(resetURL, url.Values{
		"member_id": []string{"2342"},
		// dont need to setup csrf on admin tests
	})
	a.Equal(http.StatusOK, resp.Code)

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	require.NoError(t, err)

	gotResetURL, has := doc.Find("#password-reset-link").Attr("href")
	a.True(has, "should have an href")

	wantResetURL := ts.URLTo(router.MembersChangePassword, "token", testToken)
	a.Equal(wantResetURL.String(), gotResetURL)
}
