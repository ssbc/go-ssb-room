package admin

import (
	"errors"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/gorilla/csrf"
	"github.com/russross/blackfriday/v2"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"go.mindeco.de/http/render"
)

type noticeHandler struct {
	r *render.Renderer

	db admindb.NoticesService
}

func (nh noticeHandler) edit(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	n, err := nh.db.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, admindb.ErrNotFound) {
			http.Redirect(rw, req, redirectTo, http.StatusFound)
			return nil, ErrRedirected
		}
		return nil, err
	}

	// https://github.com/russross/blackfriday/issues/575
	fixedContent := strings.Replace(n.Content, "\r\n", "\n", -1)

	contentBytes := []byte(fixedContent)
	preview := blackfriday.Run(contentBytes)

	return map[string]interface{}{
		"Notice":         n,
		"ContentPreview": template.HTML(preview),
		// "Debug":          string(preview),
		// "DebugHex":       hex.Dump(contentBytes),
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (nh noticeHandler) save(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "form data", Details: err}
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	redirect := req.FormValue("redirect")
	if redirect == "" {
		redirect = "/"
	}

	var n admindb.Notice
	n.ID, err = strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "id", Details: err}
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	n.Title = req.FormValue("title")

	n.Content = req.FormValue("content")
	// https://github.com/russross/blackfriday/issues/575
	n.Content = strings.Replace(n.Content, "\r\n", "\n", -1)

	err = nh.db.Save(req.Context(), &n)
	if err != nil {
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	// TODO: update langauge

	http.Redirect(rw, req, redirect, http.StatusTemporaryRedirect)
}
