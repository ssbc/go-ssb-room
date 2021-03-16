package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	"github.com/stretchr/testify/assert"
)

// Verifies that the notice.go save handler is like, actually, called.
func TestNoticeSaveActuallyCalled(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the urlTo helper (constructs urls for us!)
	urlTo := web.NewURLTo(ts.Router)

	id := []string{"1"}
	title := []string{"SSB Breaking News: This Test Is Great"}
	content := []string{"Absolutely Thrilling Content"}
	language := []string{"en-GB"}

	// POST a correct request to the save handler, and verify that the save was handled using the mock database)
	u := urlTo(router.AdminNoticeSave)
	formValues := url.Values{"id": id, "title": title, "content": content, "language": language}
	resp := ts.Client.PostForm(u.String(), formValues)
	a.Equal(http.StatusSeeOther, resp.Code, "POST should work")
	a.Equal(1, ts.NoticeDB.SaveCallCount(), "noticedb should have saved after POST completed")
}

// Verifies that the notices.go:save handler refuses requests missing required parameters
func TestNoticeSaveRefusesIncomplete(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the urlTo helper (constructs urls for us!)
	urlTo := web.NewURLTo(ts.Router)

	// notice values we are selectively omitting in the tests below
	id := []string{"1"}
	title := []string{"SSB Breaking News: This Test Is Great"}
	content := []string{"Absolutely Thrilling Content"}
	language := []string{"pt"}

	/* save without id */
	u := urlTo(router.AdminNoticeSave)
	emptyParams := url.Values{}
	resp := ts.Client.PostForm(u.String(), emptyParams)
	a.Equal(http.StatusInternalServerError, resp.Code, "saving without id should not work")

	/* save without title */
	formValues := url.Values{"id": id, "content": content, "language": language}
	resp = ts.Client.PostForm(u.String(), formValues)
	a.Equal(http.StatusInternalServerError, resp.Code, "saving without title should not work")

	/* save without content */
	formValues = url.Values{"id": id, "title": title, "language": language}
	resp = ts.Client.PostForm(u.String(), formValues)
	a.Equal(http.StatusInternalServerError, resp.Code, "saving without content should not work")

	/* save without language */
	formValues = url.Values{"id": id, "title": title, "content": content}
	resp = ts.Client.PostForm(u.String(), formValues)
	a.Equal(http.StatusInternalServerError, resp.Code, "saving without language should not work")

	a.Equal(0, ts.NoticeDB.SaveCallCount(), "noticedb should never save incomplete requests")
}

// Verifies that /translation/add only accepts POST requests
func TestNoticeAddLanguageOnlyAllowsPost(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the urlTo helper (constructs urls for us!)
	urlTo := web.NewURLTo(ts.Router)

	// verify that a GET request is no bueno
	u := urlTo(router.AdminNoticeAddTranslation, "name", roomdb.NoticeNews.String())
	_, resp := ts.Client.GetHTML(u.String())
	a.Equal(http.StatusMethodNotAllowed, resp.Code, "GET should not be allowed for this route")

	// next up, we verify that a correct POST request actually works:
	id := []string{"1"}
	title := []string{"Bom Dia! SSB Breaking News: This Test Is Great"}
	content := []string{"conteúdo muito bom"}
	language := []string{"pt"}

	formValues := url.Values{"name": []string{roomdb.NoticeNews.String()}, "id": id, "title": title, "content": content, "language": language}
	resp = ts.Client.PostForm(u.String(), formValues)
	a.Equal(http.StatusTemporaryRedirect, resp.Code)
}

// Verifies that the "add a translation" page contains all the required form fields (id/title/content/language)
func TestNoticeDraftLanguageIncludesAllFields(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the urlTo helper (constructs urls for us!)
	urlTo := web.NewURLTo(ts.Router)

	// to test translations we first need to add a notice to the notice mockdb
	notice := roomdb.Notice{
		ID:       1,
		Title:    "News",
		Content:  "Breaking News: This Room Has News",
		Language: "en-GB",
	}
	// make sure we return a notice when accessing pinned notices (which are the only notices with translations at writing (2021-03-11)
	ts.PinnedDB.GetReturns(&notice, nil)

	u := urlTo(router.AdminNoticeDraftTranslation, "name", roomdb.NoticeNews.String())
	html, resp := ts.Client.GetHTML(u.String())
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
	// instantiate the urlTo helper (constructs urls for us!)
	urlTo := web.NewURLTo(ts.Router)

	// Create mock notice data to operate on
	notice := roomdb.Notice{
		ID:       1,
		Title:    "News",
		Content:  "Breaking News: This Room Has News",
		Language: "en-GB",
	}
	ts.NoticeDB.GetByIDReturns(notice, nil)

	u := urlTo(router.AdminNoticeEdit, "id", 1)
	html, resp := ts.Client.GetHTML(u.String())
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
