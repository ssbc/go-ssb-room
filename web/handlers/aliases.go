package handlers

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gorilla/mux"
	"go.mindeco.de/http/render"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/aliases"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
)

// aliasHandler implements the public resolve endpoint for HTML and JSON requests.
type aliasHandler struct {
	r *render.Renderer

	db roomdb.AliasService

	muxrpcHostAndPort string
	roomID            refs.FeedRef
}

func (a aliasHandler) resolve(rw http.ResponseWriter, req *http.Request) {
	respEncoding := req.URL.Query().Get("encoding")

	var ar aliasResponder
	switch respEncoding {
	case "json":
		ar = newAliasJSONResponder(rw)
	default:
		ar = newAliasHTMLResponder(a.r, rw, req)
	}

	ar.UpdateRoomInfo(a.muxrpcHostAndPort, a.roomID)

	name := mux.Vars(req)["alias"]
	if name == "" && !aliases.IsValid(name) {
		ar.SendError(fmt.Errorf("invalid alias"))
		return
	}

	alias, err := a.db.Resolve(req.Context(), name)
	if err != nil {
		ar.SendError(err)
		return
	}

	ar.SendConfirmation(alias)
}

// aliasResponder is supposed to handle different encoding types transparently.
// It either sends the signed alias confirmation or an error.
type aliasResponder interface {
	SendConfirmation(roomdb.Alias)
	SendError(error)

	UpdateRoomInfo(hostAndPort string, roomID refs.FeedRef)
}

// aliasJSONResponse dictates the field names and format of the JSON response for the alias web endpoint
type aliasJSONResponse struct {
	Status    string `json:"status"`
	Address   string `json:"address"`
	RoomID    string `json:"roomId"`
	UserID    string `json:"userId"`
	Alias     string `json:"alias"`
	Signature string `json:"signature"`
}

// handles JSON responses
type aliasJSONResponder struct {
	enc *json.Encoder

	roomID        refs.FeedRef
	multiservAddr string
}

func newAliasJSONResponder(rw http.ResponseWriter) aliasResponder {
	return &aliasJSONResponder{
		enc: json.NewEncoder(rw),
	}
}

func (json *aliasJSONResponder) UpdateRoomInfo(hostAndPort string, roomID refs.FeedRef) {
	json.roomID = roomID

	roomPubKey := base64.StdEncoding.EncodeToString(roomID.PubKey())
	json.multiservAddr = fmt.Sprintf("net:%s~shs:%s", hostAndPort, roomPubKey)
}

func (json aliasJSONResponder) SendConfirmation(alias roomdb.Alias) {
	var resp = aliasJSONResponse{
		Status:    "successfull",
		RoomID:    json.roomID.Ref(),
		Address:   json.multiservAddr,
		Alias:     alias.Name,
		UserID:    alias.Feed.Ref(),
		Signature: base64.StdEncoding.EncodeToString(alias.Signature),
	}
	json.enc.Encode(resp)
}

func (json aliasJSONResponder) SendError(err error) {
	json.enc.Encode(struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}{"error", err.Error()})
}

// handles HTML responses
type aliasHTMLResponder struct {
	renderer *render.Renderer
	rw       http.ResponseWriter
	req      *http.Request

	roomID        refs.FeedRef
	multiservAddr string
}

func newAliasHTMLResponder(r *render.Renderer, rw http.ResponseWriter, req *http.Request) aliasResponder {
	return &aliasHTMLResponder{
		renderer: r,
		rw:       rw,
		req:      req,
	}
}

func (html *aliasHTMLResponder) UpdateRoomInfo(hostAndPort string, roomID refs.FeedRef) {
	html.roomID = roomID

	roomPubKey := base64.StdEncoding.EncodeToString(roomID.PubKey())
	html.multiservAddr = fmt.Sprintf("net:%s~shs:%s", hostAndPort, roomPubKey)
}

func (html aliasHTMLResponder) SendConfirmation(alias roomdb.Alias) {
	err := html.renderer.Render(html.rw, html.req, "aliases-resolved.html", http.StatusOK, struct {
		Alias roomdb.Alias

		RoomAddr string
	}{alias, html.multiservAddr})
	if err != nil {
		log.Println("alias-resolve render errr:", err)
	}
}

func (html aliasHTMLResponder) SendError(err error) {
	html.renderer.Error(html.rw, html.req, http.StatusInternalServerError, err)
}
