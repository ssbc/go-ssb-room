package admin

import (
	"fmt"
	"net/http"
	// "net/url"
	"testing"

	// "github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
)

/* TODO: 500s for some reason? */
func TestNoticeEditFormIncludesAllFields(t *testing.T) {
	ts := newSession(t)
	// instantiate the urlTo helper (constructs urls for us!)
	urlTo := web.NewURLTo(ts.Router)

	checkFormInputs := func(t *testing.T, url string) {
		html, resp := ts.Client.GetHTML(url)
		fmt.Println(html.Html())
		a := assert.New(t)
		/* TODO: continue test by checking for 1) forms, and 2) their required input fields */
		a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
		// Phrased these tests this way (multiple tests checking #) to present less confusion if, somehow, the inputs somehow end up being more than 1 :)
		titleInputs := html.Find(`input[name="title"]`).Length()
		a.True(titleInputs > 0, "Title input is missing")
		a.True(titleInputs == 1, "Expected only one title input (there were several)")

		idInput := html.Find(`input[name="id"]`)
		idType, idHasType := idInput.Attr("type")
		idInputs := idInput.Length()
		a.True(idInputs > 0, "ID input is missing")
		a.True(idInputs == 1, "Expected only one id input (there were several)")
		a.True(idHasType && idType == "hidden", "Expected id input to be of type hidden")
	}
	// Construct notice url to edit
	url := urlTo(router.AdminNoticeEdit, "id", 0).String()
	checkFormInputs(t, url)
	// Create mock notice data to operate on
	// notice := admindb.Notice{
	// 	ID:       1,
	// 	Title:    "News",
	// 	Content:  "Breaking News: This Room Has News",
	// 	Language: "en",
	// }
}

/* TODO: use this test as an example for pair programming 2021-03-09 */
// Assumption: I can use all the exported routes for my tests
// Reality: I can only use the routes associated with the 'correct' testing setup function
// (setup for all, newSession for admin)

// TestEditButtonVisibleForAdmin specifically tests if the regular notice view at /notice/show?id=<x>
// displays an edit button for admin users
func TestNoticeEditButtonVisibleForAdmin(t *testing.T) {
	ts := newSession(t)
	// a := assert.New(t)

	urlTo := web.NewURLTo(ts.Router)

	// Construct notice url to edit
	baseurl := urlTo(router.CompleteNoticeShow, "id", 3)

	t.Log(baseurl.String())

	// url := fmt.Sprintf("%s?id=%d", baseurl.String(), 1)
	//
	// html, resp := ts.Client.GetHTML(url)
	// fmt.Println(html.Html())
	// a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
}
