// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package admin

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	refs "github.com/ssbc/go-ssb-refs"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web/router"
	"github.com/ssbc/go-ssb-room/v2/web/webassert"
)

func TestDeniedKeysEmpty(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url := ts.URLTo(router.AdminDeniedKeysOverview)

	html, resp := ts.Client.GetHTML(url)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDeniedKeysWelcome"},
		{"title", "AdminDeniedKeysTitle"},
		{"#DeniedKeysCount", "MemberCountPlural"},
	})
}

func TestDeniedKeysAddDisabledInterface(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeRestricted, nil)

	listURL := ts.URLTo(router.AdminDeniedKeysOverview)

	ts.User = roomdb.Member{
		ID:   1234,
		Role: roomdb.RoleAdmin,
	}

	// check basic form
	html, resp := ts.Client.GetHTML(listURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	formSelection := html.Find("form#add-entry")
	a.EqualValues(1, formSelection.Length())

	method, ok := formSelection.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := formSelection.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminDeniedKeysAdd)
	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, formSelection, []webassert.FormElement{
		{Name: "pub_key", Type: "text"},
		{Name: "comment", Type: "text"},
	})

	// create assertion helpers
	newKey := "@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519"
	addVals := url.Values{"comment": []string{"some comment"}, "pub_key": []string{newKey}}
	totalAddCallCount := 0
	checkCanPostNewEntry := func(t *testing.T, shouldWork bool) {
		a := assert.New(t)
		r := require.New(t)
		rec := ts.Client.PostForm(addURL, addVals)

		a.Equal(http.StatusSeeOther, rec.Code)
		a.Equal(listURL.Path, rec.Header().Get("Location"))

		var wantedLabel = "ErrorNotAuthorized"
		if shouldWork {
			totalAddCallCount++
			wantedLabel = "AdminDeniedKeysAdded"

			// require call count to not panic
			r.Equal(totalAddCallCount, ts.DeniedKeysDB.AddCallCount())
			_, addedKey, addedComment := ts.DeniedKeysDB.AddArgsForCall(totalAddCallCount - 1)
			a.Equal(newKey, addedKey.String())
			a.Equal("some comment", addedComment)
		} else {
			r.Equal(totalAddCallCount, ts.DeniedKeysDB.AddCallCount())
		}

		webassert.HasFlashMessages(t, ts.Client, listURL, wantedLabel)
	}

	/* Verify that the inputs are visible/hidden depending on user roles */
	checkInputsAreDisabled := func(t *testing.T, shouldBeDisabled bool) {
		a := assert.New(t)
		html, resp := ts.Client.GetHTML(listURL)
		a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
		inputContainer := html.Find("#denied-keys-input-container")
		a.Equal(1, inputContainer.Length())
		inputs := inputContainer.Find("input")
		// pubkey, comment, submit button
		a.Equal(3, inputs.Length())
		inputs.Each(func(i int, el *goquery.Selection) {
			_, disabled := el.Attr("disabled")
			name, _ := el.Attr("name")
			a.Equal(shouldBeDisabled, disabled, "found diabled tag on element %q", name)
		})
	}

	/* test various restricted mode with the roles member, mod, admin */
	for _, mode := range roomdb.AllPrivacyModes {
		t.Run(mode.String(), func(t *testing.T) {
			ts.ConfigDB.GetPrivacyModeReturns(mode, nil)
			t.Run("role:member", func(t *testing.T) {
				ts.User = roomdb.Member{
					ID:   7331,
					Role: roomdb.RoleMember,
				}
				checkInputsAreDisabled(t, mode != roomdb.ModeCommunity)
				checkCanPostNewEntry(t, mode == roomdb.ModeCommunity)
			})
			t.Run("role:moderator", func(t *testing.T) {
				ts.User = roomdb.Member{
					ID:   9001,
					Role: roomdb.RoleModerator,
				}
				checkInputsAreDisabled(t, false)
				checkCanPostNewEntry(t, true)
			})
			t.Run("role:admin", func(t *testing.T) {
				ts.User = roomdb.Member{
					ID:   1234,
					Role: roomdb.RoleAdmin,
				}
				checkInputsAreDisabled(t, false)
				checkCanPostNewEntry(t, true)
			})
		})
	}
}

func TestDeniedKeysAdd(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	listURL := ts.URLTo(router.AdminDeniedKeysOverview)

	html, resp := ts.Client.GetHTML(listURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	formSelection := html.Find("form#add-entry")
	a.EqualValues(1, formSelection.Length())

	method, ok := formSelection.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := formSelection.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminDeniedKeysAdd)
	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, formSelection, []webassert.FormElement{
		{Name: "pub_key", Type: "text"},
		{Name: "comment", Type: "text"},
	})

	newKey := "@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519"
	addVals := url.Values{
		"comment": []string{"some comment"},
		// just any key that looks valid
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)

	overview := ts.URLTo(router.AdminDeniedKeysOverview)
	a.Equal(overview.Path, rec.Header().Get("Location"))
	webassert.HasFlashMessages(t, ts.Client, overview, "AdminDeniedKeysAdded")

	a.Equal(1, ts.DeniedKeysDB.AddCallCount())
	_, addedKey, addedComment := ts.DeniedKeysDB.AddArgsForCall(0)
	a.Equal(newKey, addedKey.String())
	a.Equal("some comment", addedComment)
}

func TestDeniedKeysDontAddInvalid(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	addURL := ts.URLTo(router.AdminDeniedKeysAdd)

	newKey := "@some-garbage"
	addVals := url.Values{
		"comment": []string{"some-comment"},
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)

	a.Equal(0, ts.DeniedKeysDB.AddCallCount(), "did not call add")

	listURL := ts.URLTo(router.AdminDeniedKeysOverview)
	res := rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	webassert.HasFlashMessages(t, ts.Client, listURL, "ErrorBadRequest")
}

func TestDeniedKeys(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	fakeFeed, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte{0}, 32), refs.RefAlgoFeedSSB1)
	if err != nil {
		t.Error(err)
	}

	oneThreeOneTwoFeed, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("1312"), 8), refs.RefAlgoFeedSSB1)
	if err != nil {
		t.Error(err)
	}

	acabFeed, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte("acab"), 8), refs.RefAlgoFeedSSB1)
	if err != nil {
		t.Error(err)
	}

	lst := []roomdb.ListEntry{
		{ID: 1, PubKey: fakeFeed},
		{ID: 2, PubKey: oneThreeOneTwoFeed},
		{ID: 3, PubKey: acabFeed},
	}
	ts.DeniedKeysDB.ListReturns(lst, nil)

	deniedOverviewURL := ts.URLTo(router.AdminDeniedKeysOverview)

	html, resp := ts.Client.GetHTML(deniedOverviewURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDeniedKeysWelcome"},
		{"title", "AdminDeniedKeysTitle"},
		{"#DeniedKeysCount", "MemberCountPlural"},
	})

	a.EqualValues(html.Find("#theList li").Length(), 3)

	feed, err := refs.NewFeedRefFromBytes(bytes.Repeat([]byte{1}, 32), refs.RefAlgoFeedSSB1)
	if err != nil {
		t.Error(err)
	}
	lst = []roomdb.ListEntry{
		{ID: 666, PubKey: feed},
	}
	ts.DeniedKeysDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML(deniedOverviewURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminDeniedKeysWelcome"},
		{"title", "AdminDeniedKeysTitle"},
		{"#DeniedKeysCount", "MemberCountSingular"},
	})

	elems := html.Find("#theList li")
	a.EqualValues(elems.Length(), 1)

	// check for link to remove confirm link
	link, yes := elems.ContentsFiltered("a").Attr("href")
	a.True(yes, "a-tag has href attribute")
	wantLink := ts.URLTo(router.AdminDeniedKeysRemoveConfirm, "id", 666)
	a.Equal(wantLink.String(), link)
}

func TestDeniedKeysRemoveConfirmation(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testKey, err := refs.ParseFeedRef("@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519")
	a.NoError(err)
	testEntry := roomdb.ListEntry{ID: 666, PubKey: testKey}
	ts.DeniedKeysDB.GetByIDReturns(testEntry, nil)

	urlRemoveConfirm := ts.URLTo(router.AdminDeniedKeysRemoveConfirm, "id", 3)

	html, resp := ts.Client.GetHTML(urlRemoveConfirm)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(testKey.String(), html.Find("pre#verify").Text(), "has the key for verification")

	form := html.Find("form#confirm")

	method, ok := form.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := form.Attr("action")
	a.True(ok, "form has action set")

	addURL := ts.URLTo(router.AdminDeniedKeysRemove)
	a.Equal(addURL.String(), action)

	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "id", Type: "hidden", Value: "666"},
	})
}

func TestDeniedKeysRemove(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	urlRemove := ts.URLTo(router.AdminDeniedKeysRemove)

	ts.DeniedKeysDB.RemoveIDReturns(nil)

	addVals := url.Values{"id": []string{"666"}}
	rec := ts.Client.PostForm(urlRemove, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)

	overview := ts.URLTo(router.AdminDeniedKeysOverview)
	a.Equal(overview.Path, rec.Header().Get("Location"))
	webassert.HasFlashMessages(t, ts.Client, overview, "AdminDeniedKeysRemoved")

	a.Equal(1, ts.DeniedKeysDB.RemoveIDCallCount())
	_, theID := ts.DeniedKeysDB.RemoveIDArgsForCall(0)
	a.EqualValues(666, theID)

	// now for unknown ID
	ts.DeniedKeysDB.RemoveIDReturns(roomdb.ErrNotFound)
	addVals = url.Values{"id": []string{"667"}}
	rec = ts.Client.PostForm(urlRemove, addVals)
	a.Equal(http.StatusSeeOther, rec.Code)
	a.Equal(overview.Path, rec.Header().Get("Location"))
	webassert.HasFlashMessages(t, ts.Client, overview, "ErrorNotFound")
}

func TestDeniedKeysRemovalRights(t *testing.T) {
	ts := newSession(t)

	// check disabled remove button on list page
	pubKey, err := generatePubKey()
	if err != nil {
		t.Error(err)
	}
	ts.DeniedKeysDB.ListReturns([]roomdb.ListEntry{
		{ID: 666, PubKey: pubKey, Comment: "test-entry"},
	}, nil)

	urlRemoveConfirm := ts.URLTo(router.AdminDeniedKeysRemoveConfirm, "id", "666").String()
	listURL := ts.URLTo(router.AdminDeniedKeysOverview)
	listShouldShowTheRemoveButtonAsWorking := func(shouldWork bool) func(t *testing.T) {
		return func(t *testing.T) {
			a := assert.New(t)
			html, resp := ts.Client.GetHTML(listURL)
			a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
			linktToConfirm := html.Find("#theList li a")
			a.Equal(1, linktToConfirm.Length())
			// pubkey, comment, submit button
			linkText, has := linktToConfirm.Attr("href")
			a.True(has, "anchor should have a href in any case")
			if shouldWork {
				a.Equal(urlRemoveConfirm, linkText)
			} else {
				a.True(linktToConfirm.HasClass("line-through"), "should have strikethrogh class")
				a.Equal("#", linkText)
			}
		}
	}

	// check who can actually remove
	ts.DeniedKeysDB.RemoveIDReturns(nil)
	urlRemove := ts.URLTo(router.AdminDeniedKeysRemove)
	removeVals := url.Values{"id": []string{"666"}}

	totalRemoveCallCount := 0
	removeFromListShouldWork := func(works bool) func(t *testing.T) {
		return func(t *testing.T) {
			a := assert.New(t)
			r := require.New(t)
			rec := ts.Client.PostForm(urlRemove, removeVals)
			a.Equal(http.StatusSeeOther, rec.Code, "unexpected exit code %s", rec.Result().Status)
			if works {
				totalRemoveCallCount++
				_, userID := ts.DeniedKeysDB.RemoveIDArgsForCall(totalRemoveCallCount - 1)
				a.EqualValues(666, userID)
				webassert.HasFlashMessages(t, ts.Client, listURL, "AdminDeniedKeysRemoved")
			} else {
				webassert.HasFlashMessages(t, ts.Client, listURL, "ErrorNotAuthorized")
			}
			r.Equal(totalRemoveCallCount, ts.DeniedKeysDB.RemoveIDCallCount())
		}
	}

	memKey, err := generatePubKey()
	if err != nil {
		t.Error(err)
	}

	modKey, err := generatePubKey()
	if err != nil {
		t.Error(err)
	}

	adminKey, err := generatePubKey()
	if err != nil {
		t.Error(err)
	}

	// the users who will execute the action
	memberUser := roomdb.Member{
		ID:     7331,
		Role:   roomdb.RoleMember,
		PubKey: memKey,
	}
	modUser := roomdb.Member{
		ID:     9001,
		Role:   roomdb.RoleModerator,
		PubKey: modKey,
	}
	adminUser := roomdb.Member{
		ID:     1337,
		Role:   roomdb.RoleAdmin,
		PubKey: adminKey,
	}

	/* test various restricted mode with the roles member, mod, admin */
	for _, mode := range roomdb.AllPrivacyModes {
		ts.ConfigDB.GetPrivacyModeReturns(mode, nil)

		// members can remove entries only in community mode
		ts.User = memberUser
		t.Run(mode.String()+" member sees link working", listShouldShowTheRemoveButtonAsWorking(mode == roomdb.ModeCommunity))
		t.Run(mode.String()+" member can actually remove", removeFromListShouldWork(mode == roomdb.ModeCommunity))

		// mods & admins can always invite
		ts.User = modUser
		t.Run(mode.String()+" mod sees link working", listShouldShowTheRemoveButtonAsWorking(true))
		t.Run(mode.String()+" mod can actually remove", removeFromListShouldWork(true))

		ts.User = adminUser
		t.Run(mode.String()+" admin sees link working", listShouldShowTheRemoveButtonAsWorking(true))
		t.Run(mode.String()+" admin sees link working", removeFromListShouldWork(true))
	}
}
