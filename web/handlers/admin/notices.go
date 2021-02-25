package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/ssb-ngi-pointer/go-ssb-room/web/router"

	"github.com/gorilla/csrf"
	"github.com/russross/blackfriday/v2"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"go.mindeco.de/http/render"
)

type noticeHandler struct {
	r *render.Renderer

	noticeDB admindb.NoticesService
	pinnedDB admindb.PinnedNoticesService
}

func (nh noticeHandler) draftTranslation(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	pinnedName := req.URL.Query().Get("name")

	if !admindb.PinnedNoticeName(pinnedName).Valid() {
		return nil, weberrors.ErrBadRequest{Where: "pinnedName", Details: fmt.Errorf("invalid pinned notice name")}
	}

	return map[string]interface{}{
		"SubmitAction":   router.AdminNoticeAddTranslation,
		"PinnedName":     pinnedName,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (nh noticeHandler) addTranslation(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "form data", Details: err}
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	pinnedName := admindb.PinnedNoticeName(req.FormValue("name"))
	if !pinnedName.Valid() {
		err := weberrors.ErrBadRequest{Where: "name", Details: fmt.Errorf("invalid pinned notice name")}
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	var n admindb.Notice
	n.Title = req.FormValue("title")

	// TODO: validate languages properly
	n.Language = req.FormValue("language")
	if n.Language == "" {
		err := weberrors.ErrBadRequest{Where: "language", Details: fmt.Errorf("language can't be empty")}
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	n.Content = req.FormValue("content")
	// https://github.com/russross/blackfriday/issues/575
	n.Content = strings.Replace(n.Content, "\r\n", "\n", -1)

	err = nh.noticeDB.Save(req.Context(), &n)
	if err != nil {
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	err = nh.pinnedDB.Set(req.Context(), pinnedName, n.ID)
	if err != nil {
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	// TODO: redirect to edit page of the new notice (need to add urlTo to handler)
	redirect := req.FormValue("redirect")
	if redirect == "" {
		redirect = "/"
	}
	http.Redirect(rw, req, redirect, http.StatusTemporaryRedirect)
}

func (nh noticeHandler) edit(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	n, err := nh.noticeDB.GetByID(req.Context(), id)
	if err != nil {
		return nil, err
	}

	// https://github.com/russross/blackfriday/issues/575
	fixedContent := strings.Replace(n.Content, "\r\n", "\n", -1)

	contentBytes := []byte(fixedContent)
	preview := blackfriday.Run(contentBytes)

	return map[string]interface{}{
		"SubmitAction":   router.AdminNoticeSave,
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

	// TODO: validate languages properly
	n.Language = req.FormValue("language")
	if n.Language == "" {
		err = weberrors.ErrBadRequest{Where: "language", Details: fmt.Errorf("language can't be empty")}
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	n.Content = req.FormValue("content")
	// https://github.com/russross/blackfriday/issues/575
	n.Content = strings.Replace(n.Content, "\r\n", "\n", -1)

	err = nh.noticeDB.Save(req.Context(), &n)
	if err != nil {
		nh.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	http.Redirect(rw, req, redirect, http.StatusTemporaryRedirect)
}