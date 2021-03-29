package admin

import (
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	"github.com/stretchr/testify/assert"
)

func TestDashoard(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	url, err := ts.Router.Get(router.AdminDashboard).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal(1, html.Find("#online").Size())
	a.Equal(1, html.Find("#members").Size())
	a.Equal(1, html.Find("#invites").Size())
	a.Equal(1, html.Find("#banned").Size())

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"title", "AdminDashboardTitle"},
	})
}
