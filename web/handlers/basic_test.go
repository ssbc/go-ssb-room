// SPDX-License-Identifier: MIT

package handlers

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
)

func TestIndex(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	url := ts.URLTo(router.CompleteIndex)

	html, resp := ts.Client.GetHTML(url)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"h1", "Default Notice Title"},
		{"title", "Default Notice Title"},
	})

	content := html.Find("p").Text()
	a.Equal("Default Notice Content", content)
}

func TestNotFound(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)

	url404, err := url.Parse("/some/random/ASDKLANZXC")
	a.NoError(err)

	html, resp := ts.Client.GetHTML(url404)
	a.Equal(http.StatusNotFound, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #404 - Not Found", found)
}
