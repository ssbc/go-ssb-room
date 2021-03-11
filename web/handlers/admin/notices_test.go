package admin

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/PuerkitoBio/goquery"
	"github.com/stretchr/testify/assert"
)
/* TODO: 
    * add a new test that makes sure that /translation/add only accepts POST 
    * add a check inside the handler proper
*/

func createTestElementCheck (t *testing.T, html *goquery.Selection) func (string, string) {
    a := assert.New(t)
    return func(tag, name string) {
        inputs := html.Find(fmt.Sprintf(`%s[name="%s"]`, tag, name)).Length()
        // phrased these tests this way (multiple tests checking #) to present less confusion if, somehow, the inputs end up being more than 1 :)
        a.True(inputs > 0, fmt.Sprintf("%s input is missing", strings.Title(name)))
        a.True(inputs < 2, fmt.Sprintf("Expected only one %s input (there were several)", name))
    }
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
    testElementExistence := createTestElementCheck(t, form)
    testElementExistence("textarea", "content")
    webassert.InputsInForm(t, form, []webassert.InputElement{
        { Name: "title" },
        { Name: "language" },
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
    testElementExistence := createTestElementCheck(t, form)

    a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
    // check for all the form elements & verify their initial contents are set correctly
    testElementExistence("textarea", "content")
    webassert.InputsInForm(t, form, []webassert.InputElement{
        { Name: "title", Value: notice.Title },
        { Name: "language", Value: notice.Language },
        { Name: "id", Value: fmt.Sprintf("%d", notice.ID), Type: "hidden" },
    })
}
