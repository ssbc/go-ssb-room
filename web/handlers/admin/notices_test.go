package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	"github.com/stretchr/testify/assert"
)

// Verifies that the notice.go save handler is like, actually, called.
func TestNoticeSaveActuallyCalled(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	id := []string{"1"}
	title := []string{"SSB Breaking News: This Test Is Great"}
	content := []string{"Absolutely Thrilling Content"}
	language := []string{"en-GB"}

	// POST a correct request to the save handler, and verify that the save was handled using the mock database)
	u := ts.URLTo(router.AdminNoticeSave)
	formValues := url.Values{"id": id, "title": title, "content": content, "language": language}
	resp := ts.Client.PostForm(u, formValues)
	a.Equal(http.StatusSeeOther, resp.Code, "POST should work")
	a.Equal(1, ts.NoticeDB.SaveCallCount(), "noticedb should have saved after POST completed")
}

// Verifies that the notices.go:save handler refuses requests missing required parameters
func TestNoticeSaveRefusesIncomplete(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	// notice values we are selectively omitting in the tests below
	id := []string{"1"}
	title := []string{"SSB Breaking News: This Test Is Great"}
	content := []string{"Absolutely Thrilling Content"}
	language := []string{"pt"}

	/* save without id */
	u := ts.URLTo(router.AdminNoticeSave)
	emptyParams := url.Values{}
	resp := ts.Client.PostForm(u, emptyParams)
	a.Equal(http.StatusSeeOther, resp.Code, "saving without id should not work")

	loc := resp.Header().Get("Location")

	noticesList := ts.URLTo(router.CompleteNoticeList)
	a.Equal(noticesList.String(), loc)

	// we should get noticesList here but
	// due to issue #35 we cant get /notices/list in package admin tests
	// but it doesn't really matter since the flash messages are rendered on whatever page the client goes to next
	sigh := ts.URLTo(router.AdminDashboard)

	webassert.HasFlashMessages(t, ts.Client, sigh, "ErrorBadRequest")

	/* save without title */
	formValues := url.Values{"id": id, "content": content, "language": language}
	resp = ts.Client.PostForm(u, formValues)
	a.Equal(http.StatusSeeOther, resp.Code, "saving without title should not work")
	webassert.HasFlashMessages(t, ts.Client, sigh, "ErrorBadRequest")

	/* save without content */
	formValues = url.Values{"id": id, "title": title, "language": language}
	resp = ts.Client.PostForm(u, formValues)
	a.Equal(http.StatusSeeOther, resp.Code, "saving without content should not work")
	webassert.HasFlashMessages(t, ts.Client, sigh, "ErrorBadRequest")

	/* save without language */
	formValues = url.Values{"id": id, "title": title, "content": content}
	resp = ts.Client.PostForm(u, formValues)
	a.Equal(http.StatusSeeOther, resp.Code, "saving without language should not work")
	webassert.HasFlashMessages(t, ts.Client, sigh, "ErrorBadRequest")

	a.Equal(0, ts.NoticeDB.SaveCallCount(), "noticedb should never save incomplete requests")
}

// Verifies that /translation/add only accepts POST requests
func TestNoticeAddLanguageOnlyAllowsPost(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the ts.URLTo helper (constructs urls for us!)

	// verify that a GET request is no bueno
	u := ts.URLTo(router.AdminNoticeAddTranslation, "name", roomdb.NoticeNews.String())
	_, resp := ts.Client.GetHTML(u)
	a.Equal(http.StatusBadRequest, resp.Code, "GET should not be allowed for this route")

	// next up, we verify that a correct POST request actually works:
	id := []string{"1"}
	title := []string{"Bom Dia! SSB Breaking News: This Test Is Great"}
	content := []string{"conteúdo muito bom"}
	language := []string{"pt"}

	formValues := url.Values{"name": []string{roomdb.NoticeNews.String()}, "id": id, "title": title, "content": content, "language": language}
	resp = ts.Client.PostForm(u, formValues)
	a.Equal(http.StatusSeeOther, resp.Code)
	webassert.HasFlashMessages(t, ts.Client, ts.URLTo(router.AdminDashboard), "NoticeUpdated")
}

// Verifies that the "add a translation" page contains all the required form fields (id/title/content/language)
func TestNoticeDraftLanguageIncludesAllFields(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the ts.URLTo helper (constructs urls for us!)

	// to test translations we first need to add a notice to the notice mockdb
	notice := roomdb.Notice{
		ID:       1,
		Title:    "News",
		Content:  "Breaking News: This Room Has News",
		Language: "en-GB",
	}
	// make sure we return a notice when accessing pinned notices (which are the only notices with translations at writing (2021-03-11)
	ts.PinnedDB.GetReturns(&notice, nil)

	u := ts.URLTo(router.AdminNoticeDraftTranslation, "name", roomdb.NoticeNews.String())
	html, resp := ts.Client.GetHTML(u)
	form := html.Find("form")
	a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
	// FormElement defaults to input if tag omitted
	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "title"},
		{Name: "language"},
		{Tag: "textarea", Name: "content"},
	})
}

func TestNoticeEditFormIncludesAllFields(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the ts.URLTo helper (constructs urls for us!)

	// Create mock notice data to operate on
	notice := roomdb.Notice{
		ID:       1,
		Title:    "News",
		Content:  "Breaking News: This Room Has News",
		Language: "en-GB",
	}
	ts.NoticeDB.GetByIDReturns(notice, nil)

	u := ts.URLTo(router.AdminNoticeEdit, "id", 1)
	html, resp := ts.Client.GetHTML(u)
	form := html.Find("form")

	a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
	// check for all the form elements & verify their initial contents are set correctly
	// FormElement defaults to input if tag omitted
	webassert.ElementsInForm(t, form, []webassert.FormElement{
		{Name: "title", Value: notice.Title},
		{Name: "language", Value: notice.Language},
		{Name: "id", Value: fmt.Sprintf("%d", notice.ID), Type: "hidden"},
		{Tag: "textarea", Name: "content"},
	})
}

func TestNoticesRoleRightsEditing(t *testing.T) {
	ts := newSession(t)

	dashboardURL := ts.URLTo(router.AdminDashboard)
	editURL := ts.URLTo(router.AdminNoticeEdit, "id", 1)
	saveURL := ts.URLTo(router.AdminNoticeSave)

	formValues := url.Values{
		"id":       []string{"1"},
		"title":    []string{"SSB Breaking News: This Test Is Great"},
		"content":  []string{"Absolutely Thrilling Content"},
		"language": []string{"en-GB"},
	}

	canSeeEditForm := func(t *testing.T, shouldWork bool) {
		a := assert.New(t)

		doc, resp := ts.Client.GetHTML(editURL)

		if shouldWork {
			a.Equal(http.StatusOK, resp.Code, "unexpected status code")
			form := doc.Find("form")
			action, has := form.Attr("action")
			a.True(has, "no action on a form?!")
			a.Equal(saveURL.String(), action)
		} else {
			a.Equal(http.StatusSeeOther, resp.Code, "unexpected status code")
			webassert.HasFlashMessages(t, ts.Client, dashboardURL, "ErrorNotAuthorized")
		}
	}

	totalSaveCallCount := 0
	canSaveNotice := func(t *testing.T, shouldWork bool) {
		a := assert.New(t)

		// POST a correct request to the save handler, and verify that the save was handled using the mock database)
		resp := ts.Client.PostForm(saveURL, formValues)
		a.Equal(http.StatusSeeOther, resp.Code, "should have redirected")

		var wantLabel string
		if shouldWork {
			totalSaveCallCount++
			wantLabel = "NoticeUpdated"
		} else {
			wantLabel = "ErrorNotAuthorized"
		}
		webassert.HasFlashMessages(t, ts.Client, dashboardURL, wantLabel)
		a.Equal(totalSaveCallCount, ts.NoticeDB.SaveCallCount(), "call count missmatch")
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

	/* test invite creation under various restricted mode with the roles member, mod, admin */
	for _, mode := range roomdb.AllPrivacyModes {
		t.Run(mode.String(), func(t *testing.T) {
			ts.ConfigDB.GetPrivacyModeReturns(mode, nil)

			// members can only invite in community rooms
			t.Run("members", func(t *testing.T) {
				ts.User = memberUser
				canSeeEditForm(t, mode == roomdb.ModeCommunity)
				canSaveNotice(t, mode == roomdb.ModeCommunity)
			})

			// mods & admins can always invite
			t.Run("mods", func(t *testing.T) {
				ts.User = modUser
				canSeeEditForm(t, true)
				canSaveNotice(t, true)
			})

			t.Run("admins", func(t *testing.T) {
				ts.User = adminUser
				canSeeEditForm(t, true)
				canSaveNotice(t, true)
			})
		})
	}
}

func TestNoticesRoleRightsAddingTranslation(t *testing.T) {
	ts := newSession(t)

	dashboardURL := ts.URLTo(router.AdminDashboard)
	draftTrURL := ts.URLTo(router.AdminNoticeDraftTranslation, "name", "NoticeNews")
	addTrURL := ts.URLTo(router.AdminNoticeAddTranslation)

	formValues := url.Values{
		"id":       []string{"1"},
		"title":    []string{"SSB Breaking News: This Test Is Great"},
		"content":  []string{"Absolutely Thrilling Content"},
		"language": []string{"en-GB"},

		"name": []string{"NoticeNews"},
	}

	canSeeAddTranslationForm := func(t *testing.T, shouldWork bool) {
		a := assert.New(t)

		doc, resp := ts.Client.GetHTML(draftTrURL)

		if shouldWork {
			a.Equal(http.StatusOK, resp.Code, "unexpected status code")
			form := doc.Find("form")
			action, has := form.Attr("action")
			a.True(has, "no action on a form?!")
			a.Equal(addTrURL.String(), action)
		} else {
			a.Equal(http.StatusSeeOther, resp.Code, "unexpected status code")
			webassert.HasFlashMessages(t, ts.Client, dashboardURL, "ErrorNotAuthorized")
		}
	}

	totalAddCallCount := 0
	canAddNewTranslation := func(t *testing.T, shouldWork bool) {
		a := assert.New(t)

		// POST a correct request to the save handler, and verify that the save was handled using the mock database)
		resp := ts.Client.PostForm(addTrURL, formValues)
		a.Equal(http.StatusSeeOther, resp.Code, "should have redirected")

		var wantLabel string
		if shouldWork {
			totalAddCallCount++
			wantLabel = "NoticeUpdated"
		} else {
			wantLabel = "ErrorNotAuthorized"
		}
		webassert.HasFlashMessages(t, ts.Client, dashboardURL, wantLabel)
		a.Equal(totalAddCallCount, ts.PinnedDB.SetCallCount(), "call count missmatch")
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

	/* test invite creation under various restricted mode with the roles member, mod, admin */
	for _, mode := range roomdb.AllPrivacyModes {
		t.Run(mode.String(), func(t *testing.T) {
			ts.ConfigDB.GetPrivacyModeReturns(mode, nil)

			// members can only invite in community rooms
			t.Run("members", func(t *testing.T) {
				ts.User = memberUser
				canSeeAddTranslationForm(t, mode == roomdb.ModeCommunity)
				canAddNewTranslation(t, mode == roomdb.ModeCommunity)
			})

			// mods & admins can always invite
			t.Run("mods", func(t *testing.T) {
				ts.User = modUser
				canSeeAddTranslationForm(t, true)
				canAddNewTranslation(t, true)
			})

			t.Run("admins", func(t *testing.T) {
				ts.User = adminUser
				canSeeAddTranslationForm(t, true)
				canAddNewTranslation(t, true)
			})
		})
	}
}
