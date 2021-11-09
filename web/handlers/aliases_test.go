// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"
	refs "go.mindeco.de/ssb-refs"
)

func TestAliasResolve(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)
	r := require.New(t)

	var testAlias = roomdb.Alias{
		ID:   54321,
		Name: "test-name",
		Feed: refs.FeedRef{
			ID:   bytes.Repeat([]byte{'F'}, 32),
			Algo: "test",
		},
		Signature: bytes.Repeat([]byte{'S'}, 32),
	}
	ts.AliasesDB.ResolveReturns(testAlias, nil)

	// to construct the /alias/{name} url we need to bypass urlTo
	// (which builds ?alias=name)
	routes := router.CompleteApp()

	// default is HTML

	htmlURL, err := routes.Get(router.CompleteAliasResolve).URL("alias", testAlias.Name)
	r.NoError(err)

	t.Log("resolving", htmlURL.String())
	html, resp := ts.Client.GetHTML(htmlURL)
	a.Equal(http.StatusOK, resp.Code)

	a.Equal(testAlias.Name, html.Find("title").Text())

	// ssb-uri in href
	aliasHref, ok := html.Find("#alias-uri").Attr("href")
	a.True(ok)
	aliasURI, err := url.Parse(aliasHref)
	r.NoError(err)

	a.Equal("ssb", aliasURI.Scheme)
	a.Equal("experimental", aliasURI.Opaque)

	params := aliasURI.Query()
	a.Equal("consume-alias", params.Get("action"))
	a.Equal(testAlias.Name, params.Get("alias"))
	a.Equal(testAlias.Feed.Ref(), params.Get("userId"))
	sigData, err := base64.StdEncoding.DecodeString(params.Get("signature"))
	r.NoError(err)
	a.Equal(testAlias.Signature, sigData)
	a.Equal(ts.NetworkInfo.RoomID.Ref(), params.Get("roomId"))
	a.Equal(ts.NetworkInfo.MultiserverAddress(), params.Get("multiserverAddress"))

	// now as JSON
	jsonURL, err := routes.Get(router.CompleteAliasResolve).URL("alias", testAlias.Name)
	r.NoError(err)

	q := jsonURL.Query()
	q.Set("encoding", "json")
	jsonURL.RawQuery = q.Encode()
	t.Log("resolving", jsonURL.String())
	resp = ts.Client.GetBody(jsonURL)
	a.Equal(http.StatusOK, resp.Code)

	var ar aliasJSONResponse
	err = json.NewDecoder(resp.Body).Decode(&ar)
	r.NoError(err)
	a.Equal(testAlias.Name, ar.Alias)
	sigData2, err := base64.StdEncoding.DecodeString(ar.Signature)
	r.NoError(err)
	a.Equal(testAlias.Signature, sigData2)
	a.Equal(testAlias.Feed.Ref(), ar.UserID, "wrong user feed on response")
	a.Equal(ts.NetworkInfo.RoomID.Ref(), ar.RoomID, "wrong room feed on response")
	a.Equal(ts.NetworkInfo.MultiserverAddress(), ar.MultiserverAddress)

	/* alias resolving should not work for restricted rooms */
	ts.ConfigDB.GetPrivacyModeReturns(roomdb.ModeRestricted, nil)
	htmlURL, err = routes.Get(router.CompleteAliasResolve).URL("alias", testAlias.Name)
	r.NoError(err)
	html, resp = ts.Client.GetHTML(htmlURL)
	a.Equal(http.StatusInternalServerError, resp.Code)
}

func TestAliasResolveOnAndroidChrome(t *testing.T) {
	ts := setup(t)

	a := assert.New(t)
	r := require.New(t)

	var testAlias = roomdb.Alias{
		ID:   54321,
		Name: "test-name",
		Feed: refs.FeedRef{
			ID:   bytes.Repeat([]byte{'F'}, 32),
			Algo: "test",
		},
		Signature: bytes.Repeat([]byte{'S'}, 32),
	}
	ts.AliasesDB.ResolveReturns(testAlias, nil)

	// Mimic Android Chrome
	var uaHeader = make(http.Header)
	uaHeader.Set("User-Agent", "Mozilla/5.0 (Linux; Android 6.0.1; Nexus 5 Build/MOB30H) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/44.0.2403.133 Mobile Safari/537.36")
	ts.Client.SetHeaders(uaHeader)

	// to construct the /alias/{name} url we need to bypass urlTo
	// (which builds ?alias=name)
	routes := router.CompleteApp()

	// default is HTML

	htmlURL, err := routes.Get(router.CompleteAliasResolve).URL("alias", testAlias.Name)
	r.NoError(err)

	t.Log("resolving", htmlURL.String())
	html, resp := ts.Client.GetHTML(htmlURL)
	a.Equal(http.StatusOK, resp.Code)

	a.Equal(testAlias.Name, html.Find("title").Text())

	// ssb-uri in href
	aliasHref, ok := html.Find("#alias-uri").Attr("href")
	a.True(ok)
	aliasURI, err := url.Parse(aliasHref)
	r.NoError(err)

	a.Equal("intent", aliasURI.Scheme)
	a.Equal("experimental", aliasURI.Host)

	params := aliasURI.Query()
	a.Equal("consume-alias", params.Get("action"))
	a.Equal(testAlias.Name, params.Get("alias"))
	a.Equal(testAlias.Feed.Ref(), params.Get("userId"))
	sigData, err := base64.StdEncoding.DecodeString(params.Get("signature"))
	r.NoError(err)
	a.Equal(testAlias.Signature, sigData)
	a.Equal(ts.NetworkInfo.RoomID.Ref(), params.Get("roomId"))
	a.Equal(ts.NetworkInfo.MultiserverAddress(), params.Get("multiserverAddress"))

	frag := aliasURI.Fragment
	a.Equal("Intent;scheme=ssb;package=se.manyver;end;", frag)
}
