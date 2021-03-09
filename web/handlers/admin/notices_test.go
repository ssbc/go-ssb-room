package admin

import (
	"fmt"
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
)

/* TODO: 500s for some reason? */
func TestNoticeEditFormIncludesAllFields(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	// Construct notice url to edit
	baseurl, err := ts.Router.Get(router.AdminNoticeEdit).URL()
	a.Nil(err)
	url := fmt.Sprintf("%s?id=%d", baseurl.String(), 1)

	_, resp := ts.Client.GetHTML(url)
	/* TODO: continue test by checking for 1) forms, and 2) their required input fields */
	a.Equal(http.StatusOK, resp.Code, "Wrong HTTP status code")
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
