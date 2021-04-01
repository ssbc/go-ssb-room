package admin

import (
	"bytes"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

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
	a.Equal(addURL.Path, action)

	webassert.ElementsInForm(t, formSelection, []webassert.FormElement{
		{Name: "pub_key", Type: "text"},
	})

	newKey := "@x7iOLUcq3o+sjGeAnipvWeGzfuYgrXl8L4LYlxIhwDc=.ed25519"
	addVals := url.Values{
		// just any key that looks valid
		"pub_key": []string{newKey},
	}
	rec := ts.Client.PostForm(addURL, addVals)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)

	a.Equal(1, ts.MembersDB.AddCallCount())
	_, addedPubKey, addedRole := ts.MembersDB.AddArgsForCall(0)
	a.Equal(newKey, addedPubKey.Ref())
	a.Equal(roomdb.RoleMember, addedRole)

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
	a.Equal(http.StatusTemporaryRedirect, rec.Code)

	a.Equal(0, ts.MembersDB.AddCallCount())

	listURL := ts.URLTo(router.AdminMembersOverview)
	res := rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	doc, resp := ts.Client.GetHTML(listURL)
	a.Equal(http.StatusOK, resp.Code)

	flashes := doc.Find("#flashes-list").Children()
	a.Equal(1, flashes.Length())
	a.Equal("ErrorBadRequest", flashes.Text())

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

	membersOveriwURL := ts.URLTo(router.AdminMembersOverview)

	html, resp := ts.Client.GetHTML(membersOveriwURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminMembersWelcome"},
		{"title", "AdminMembersTitle"},
		{"#membersCount", "MemberCountPlural"},
	})

	a.EqualValues(html.Find("#theList li").Length(), 3)

	lst = []roomdb.Member{
		{ID: 666, Role: roomdb.RoleAdmin, PubKey: refs.FeedRef{ID: bytes.Repeat([]byte{1}, 32), Algo: "one"}},
	}
	ts.MembersDB.ListReturns(lst, nil)

	html, resp = ts.Client.GetHTML(membersOveriwURL)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"#welcome", "AdminMembersWelcome"},
		{"title", "AdminMembersTitle"},
		{"#membersCount", "MemberCountSingular"},
	})

	elems := html.Find("#theList li")
	a.EqualValues(elems.Length(), 1)

	// check for link to member details
	link, yes := elems.ContentsFiltered("a").Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/member?id=666", link)
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

	html, resp := ts.Client.GetHTML("/member?id=1")
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"title", "AdminMemberDetailsTitle"},
	})

	// check for SSB ID
	ssbID := html.Find("#ssb-id")
	a.Equal(feedRef.Ref(), ssbID.Text())

	// check for change-role dropdown
	roleDropdown := html.Find("#change-role")
	a.EqualValues(roleDropdown.Length(), 1)

	// check for link to resolve 1st Alias
	aliasRobertLink, yes := html.Find("#alias-list").Find("a").Eq(0).Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/alias/robert", aliasRobertLink)

	// check for link to revoke 1st Alias
	revokeRobertLink, yes := html.Find("#alias-list").Find("a").Eq(1).Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/aliases/revoke/confirm?id=11", revokeRobertLink)

	// check for link to resolve 1st Alias
	aliasBobLink, yes := html.Find("#alias-list").Find("a").Eq(2).Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/alias/bob", aliasBobLink)

	// check for link to revoke 1st Alias
	revokeBobLink, yes := html.Find("#alias-list").Find("a").Eq(3).Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/aliases/revoke/confirm?id=21", revokeBobLink)

	// check for link to Remove member link
	removeLink, yes := html.Find("#remove-member").Attr("href")
	a.True(yes, "a-tag has href attribute")
	a.Equal("/admin/members/remove/confirm?id=1", removeLink)
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
	a.Equal(addURL.Path, action)

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
	a.Equal(http.StatusTemporaryRedirect, rec.Code)

	a.Equal(1, ts.MembersDB.RemoveIDCallCount())
	_, theID := ts.MembersDB.RemoveIDArgsForCall(0)
	a.EqualValues(666, theID)

	listURL := ts.URLTo(router.AdminMembersOverview)
	// check flash message
	res := rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	doc, resp := ts.Client.GetHTML(listURL)
	a.Equal(http.StatusOK, resp.Code)

	flashes := doc.Find("#flashes-list").Children()
	a.Equal(1, flashes.Length())
	a.Equal("AdminMemberRemoved", flashes.Text())

	// now for unknown ID
	ts.MembersDB.RemoveIDReturns(roomdb.ErrNotFound)
	addVals = url.Values{"id": []string{"667"}}
	rec = ts.Client.PostForm(urlRemove, addVals)
	a.Equal(http.StatusTemporaryRedirect, rec.Code)

	// check flash message
	res = rec.Result()
	a.Equal(listURL.Path, res.Header.Get("Location"), "redirecting to overview")
	a.True(len(res.Cookies()) > 0, "got a cookie (flash msg)")

	doc, resp = ts.Client.GetHTML(listURL)
	a.Equal(http.StatusOK, resp.Code)

	flashes = doc.Find("#flashes-list").Children()
	a.Equal(1, flashes.Length())
	a.Equal("ErrorNotFound", flashes.Text())
}
