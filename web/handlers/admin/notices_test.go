package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
)

func TestNoticeAddLanguageIncludesAllFields(t *testing.T) {
    ts := newSession(t)
    a := assert.New(t)
    // instantiate the urlTo helper (constructs urls for us!)
    urlTo := web.NewURLTo(ts.Router)

    // to test translations we first need to add a notice to the notice mockdb
	notice := admindb.Notice{
		ID:       1,
		Title:    "News",
		Content:  "Breaking News: This Room Has News",
		Language: "en-GB",
	}
	ts.NoticeDB.GetByIDReturns(notice, nil)
    // the we need to pin the mocked notice
    ts.PinnedDB.GetReturns(&notice, nil)

    /* TODO: are you only supposed to add translations to pinned notices? */
    u := urlTo(router.AdminNoticeDraftTranslation, "name", admindb.NoticeNews.String())
    html, resp := ts.Client.GetHTML(u.String())
    a.Equal(http.StatusOK, resp.Code)
    fmt.Println(html.Html())
}

func TestNoticeEditFormIncludesAllFields(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)
	// instantiate the urlTo helper (constructs urls for us!)
	urlTo := web.NewURLTo(ts.Router)

	// helper function to test all form inputs that should exist on a given edit page
	checkFormInputs := func(u *url.URL) {
		html, resp := ts.Client.GetHTML(u.String())
		a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
		testElementExistence := func(tag, name string) {
			inputs := html.Find(fmt.Sprintf(`%s[name="%s"]`, tag, name)).Length()
			// phrased these tests this way (multiple tests checking #) to present less confusion if, somehow, the inputs end up being more than 1 :)
			a.True(inputs > 0, fmt.Sprintf("%s input is missing", strings.Title(name)))
			a.True(inputs == 1, fmt.Sprintf("Expected only one %s input (there were several)", name))
		}
		testElementExistence("input", "title")
		testElementExistence("input", "language") // this test will fail when converted to dropdown from single input
		testElementExistence("textarea", "content")
		testElementExistence("input", "id")

		// make sure the id input is hidden
		idInput := html.Find(`input[name="id"]`)
		idType, idHasType := idInput.Attr("type")
		a.True(idHasType && idType == "hidden", "Expected id input to be of type hidden")
	}

	// Create mock notice data to operate on
	notice := admindb.Notice{
		ID:       1,
		Title:    "News",
		Content:  "Breaking News: This Room Has News",
		Language: "en-GB",
	}
	ts.NoticeDB.GetByIDReturns(notice, nil)
	// Construct notice url to edit
	checkFormInputs(urlTo(router.AdminNoticeEdit, "id", 1))
}
