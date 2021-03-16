// SPDX-License-Identifier: MIT

package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
)

func TestIndex(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)
	r := require.New(t)

	url, err := ts.Router.Get(router.CompleteIndex).URL()
	r.Nil(err)
	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"h1", "Default Notice Title"},
		{"title", "Default Notice Title"},
	})

	content := html.Find("p").Text()
	a.Equal("Default Notice Content", content)
}

func TestAbout(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)
	r := require.New(t)

	url, err := ts.Router.Get(router.CompleteAbout).URL()
	r.Nil(err)
	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("The about page", found)
}

func TestNotFound(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	html, resp := ts.Client.GetHTML("/some/random/ASDKLANZXC")
	a.Equal(http.StatusNotFound, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #404 - Not Found", found)
}
