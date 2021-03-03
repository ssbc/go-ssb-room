package admin

import (
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/stretchr/testify/assert"
)

func TestInvitesCreateForm(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminInvitesOverview).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	assertLocalized(t, html, []localizedElement{
		{"#welcome", "AdminInvitesWelcome"},
		{"title", "AdminInvitesTitle"},
	})

	formSelection := html.Find("form#create-invite")
	a.EqualValues(1, formSelection.Length())

	method, ok := formSelection.Attr("method")
	a.True(ok, "form has method set")
	a.Equal("POST", method)

	action, ok := formSelection.Attr("action")
	a.True(ok, "form has action set")

	addURL, err := ts.Router.Get(router.AdminInvitesCreate).URL()
	a.NoError(err)

	a.Equal(addURL.String(), action)

	inputSelection := formSelection.Find("input[type=text]")
	a.EqualValues(1, inputSelection.Length())

	name, ok := inputSelection.Attr("name")
	a.True(ok, "input has a name")
	a.Equal("alias_suggestion", name, "wrong name on input field")
}
