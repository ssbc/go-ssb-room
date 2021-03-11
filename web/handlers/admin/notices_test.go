package admin

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
)
/* TODO: 
    * add a new 
    * add a check inside the handler proper
*/

// Verifies that /translation/add only accepts POST requests
func TestNoticeAddLanguageOnlyAllowsPost(t *testing.T) {
}

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
    webassert.ElementsInForm(t, form, []webassert.FormElement{
        { Tag: "input", Name: "title" },
        { Tag: "input", Name: "language" },
        { Tag: "textarea", Name: "content" },
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
    webassert.ElementsInForm(t, form, []webassert.FormElement{
        { Tag: "input", Name: "title", Value: notice.Title },
        { Tag: "input", Name: "language", Value: notice.Language },
        { Tag: "input", Name: "id", Value: fmt.Sprintf("%d", notice.ID), Type: "hidden" },
        { Tag: "textarea", Name: "content" },
    })
}
