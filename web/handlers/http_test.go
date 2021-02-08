package handlers

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/gossb-rooms/web/router"
)

func TestIndex(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)

	a := assert.New(t)
	r := require.New(t)

	url, err := testRouter.Get(router.CompleteIndex).URL()
	r.Nil(err)
	html, resp := testClient.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	a.Equal("Index landing page", html.Find("#welcome").Text())
	val, has := html.Find("#logo").Attr("src")
	a.True(has, "logo src attribute not found")
	a.Equal("/assets/img/test-hermie.png", val)
}

func TestAbout(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)
	r := require.New(t)

	url, err := testRouter.Get(router.CompleteAbout).URL()
	r.Nil(err)
	html, resp := testClient.GetHTML(url.String(), nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("The about page", found)
}

func TestNotFound(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)

	html, resp := testClient.GetHTML("/some/random/ASDKLANZXC", nil)
	a.Equal(http.StatusNotFound, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #404 - Not Found", found)
}

func TestNewsRegisterd(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)

	html, resp := testClient.GetHTML("/news/", nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	t.Log(found)
	// a.Equal("fooo", found)
}

func TestRestricted(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)

	html, resp := testClient.GetHTML("/admin/", nil)
	a.Equal(http.StatusUnauthorized, resp.Code, "wrong HTTP status code")
	found := html.Find("h1").Text()
	a.Equal("Error #401 - Unauthorized", found)
}

func TestLoginForm(t *testing.T) {
	setup(t)
	t.Cleanup(teardown)
	a := assert.New(t)

	html, resp := testClient.GetHTML(router.AuthFallbackSignInForm, nil)
	a.Equal(http.StatusOK, resp.Code, "wrong HTTP status code")
	found := html.Find("title").Text()
	a.Equal("Login Form", found)
}
