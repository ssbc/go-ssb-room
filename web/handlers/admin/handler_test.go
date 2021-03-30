package admin

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/webassert"
	"github.com/stretchr/testify/assert"
	refs "go.mindeco.de/ssb-refs"
)

func TestDashoard(t *testing.T) {
	ts := newSession(t)
	a := assert.New(t)

	testRef := refs.FeedRef{Algo: "test", ID: bytes.Repeat([]byte{0}, 16)}
	ts.RoomState.AddEndpoint(testRef, nil) // 1 online
	ts.MembersDB.CountReturns(4, nil)      // 4 members
	ts.InvitesDB.CountReturns(3, nil)      // 3 invites
	ts.DeniedKeysDB.CountReturns(2, nil)   // 2 banned

	url, err := ts.Router.Get(router.AdminDashboard).URL()
	a.Nil(err)

	html, resp := ts.Client.GetHTML(url.String())
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")

	a.Equal("1", html.Find("#online-count").Text())
	a.Equal("4", html.Find("#member-count").Text())
	a.Equal("3", html.Find("#invite-count").Text())
	a.Equal("2", html.Find("#denied-count").Text())

	webassert.Localized(t, html, []webassert.LocalizedElement{
		{"title", "AdminDashboardTitle"},
	})
}
