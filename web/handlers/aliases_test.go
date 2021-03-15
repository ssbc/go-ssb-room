package handlers

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"
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

	// default is HTML
	htmlURL, err := ts.Router.Get(router.CompleteAliasResolve).URL("alias", testAlias.Name)
	r.Nil(err)
	t.Log("resolving", htmlURL.String())
	html, resp := ts.Client.GetHTML(htmlURL.String())
	a.Equal(http.StatusOK, resp.Code)

	a.Equal(testAlias.Name, html.Find("title").Text())

	// default is HTML
	jsonURL, err := ts.Router.Get(router.CompleteAliasResolve).URL("alias", testAlias.Name)
	r.Nil(err)
	q := jsonURL.Query()
	q.Set("encoding", "json")
	jsonURL.RawQuery = q.Encode()
	t.Log("resolving", jsonURL.String())
	resp = ts.Client.GetBody(jsonURL.String())
	a.Equal(http.StatusOK, resp.Code)

	var ar aliasJSONResponse
	err = json.NewDecoder(resp.Body).Decode(&ar)
	r.NoError(err)
	a.Equal(testAlias.Name, ar.Alias)
	sigData, err := base64.StdEncoding.DecodeString(ar.Signature)
	r.NoError(err)
	a.Equal(testAlias.Signature, sigData)
	a.Equal(testAlias.Feed.Ref(), ar.UserID, "wrong user feed on response")
	a.Equal(ts.NetworkInfo.RoomID.Ref(), ar.RoomID, "wrong room feed on response")
}
