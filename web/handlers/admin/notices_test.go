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
	a.Equal(noticesList.Path, loc)

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
	a.Equal(http.StatusTemporaryRedirect, resp.Code)
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
