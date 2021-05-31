package admin

import (
	"fmt"
	"html/template"
	"net/http"
	"strconv"
	"strings"

	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/members"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/web/router"

	"github.com/gorilla/csrf"
	"github.com/russross/blackfriday/v2"
	"github.com/ssb-ngi-pointer/go-ssb-room/v2/roomdb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/v2/web/errors"
	"go.mindeco.de/http/render"
)

type noticeHandler struct {
	r       *render.Renderer
	urlTo   web.URLMaker
	flashes *weberrors.FlashHelper

	noticeDB roomdb.NoticesService
	pinnedDB roomdb.PinnedNoticesService
	roomCfg  roomdb.RoomConfig
}

func (h noticeHandler) draftTranslation(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	if _, err := members.CheckAllowed(req.Context(), h.roomCfg, members.ActionChangeNotice); err != nil {
		h.flashes.AddError(rw, req, err)
		noticesURL := h.urlTo(router.CompleteNoticeList)
		http.Redirect(rw, req, noticesURL.String(), http.StatusSeeOther)
		return nil, err
	}

	pinnedName := req.URL.Query().Get("name")
	if !roomdb.PinnedNoticeName(pinnedName).Valid() {
		return nil, weberrors.ErrBadRequest{Where: "pinnedName", Details: fmt.Errorf("invalid pinned notice name")}
	}

	return map[string]interface{}{
		"SubmitAction":   router.AdminNoticeAddTranslation,
		"PinnedName":     pinnedName,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h noticeHandler) addTranslation(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	// reply with 405 error: Method not allowed
	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "http method type", Details: fmt.Errorf("add translation only accepts POST requests, sorry!")}
		h.r.Error(rw, req, http.StatusMethodNotAllowed, err)
		return
	}

	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "form data", Details: err}
		h.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	redirect := req.FormValue("redirect")
	if redirect == "" {
		noticesURL := h.urlTo(router.CompleteNoticeList)
		redirect = noticesURL.String()
	}

	defer http.Redirect(rw, req, redirect, http.StatusSeeOther)

	if _, err := members.CheckAllowed(ctx, h.roomCfg, members.ActionChangeNotice); err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	pinnedName := roomdb.PinnedNoticeName(req.FormValue("name"))
	if !pinnedName.Valid() {
		err := weberrors.ErrBadRequest{Where: "name", Details: fmt.Errorf("invalid pinned notice name")}
		h.flashes.AddError(rw, req, err)
		return
	}

	var n roomdb.Notice
	n.Title = req.FormValue("title")
	if n.Title == "" {
		err = weberrors.ErrBadRequest{Where: "title", Details: fmt.Errorf("title can't be empty")}
		h.flashes.AddError(rw, req, err)
		return
	}

	// TODO: validate languages properly
	n.Language = req.FormValue("language")
	if n.Language == "" {
		err := weberrors.ErrBadRequest{Where: "language", Details: fmt.Errorf("language can't be empty")}
		h.flashes.AddError(rw, req, err)
		return
	}

	n.Content = req.FormValue("content")
	if n.Content == "" {
		err = weberrors.ErrBadRequest{Where: "content", Details: fmt.Errorf("content can't be empty")}
		h.flashes.AddError(rw, req, err)
		return
	}
	// https://github.com/russross/blackfriday/issues/575
	n.Content = strings.Replace(n.Content, "\r\n", "\n", -1)

	err = h.noticeDB.Save(ctx, &n)
	if err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	err = h.pinnedDB.Set(ctx, pinnedName, n.ID)
	if err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	h.flashes.AddMessage(rw, req, "NoticeUpdated")

}

func (h noticeHandler) edit(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	ctx := req.Context()

	if _, err := members.CheckAllowed(ctx, h.roomCfg, members.ActionChangeNotice); err != nil {
		h.flashes.AddError(rw, req, err)
		noticesURL := h.urlTo(router.CompleteNoticeList)
		http.Redirect(rw, req, noticesURL.String(), http.StatusSeeOther)
		return nil, err
	}

	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	n, err := h.noticeDB.GetByID(req.Context(), id)
	if err != nil {
		return nil, err
	}

	// https://github.com/russross/blackfriday/issues/575
	fixedContent := strings.Replace(n.Content, "\r\n", "\n", -1)

	contentBytes := []byte(fixedContent)
	preview := blackfriday.Run(contentBytes)

	pageData := map[string]interface{}{
		"SubmitAction":   router.AdminNoticeSave,
		"Notice":         n,
		"ContentPreview": template.HTML(preview),
		csrf.TemplateTag: csrf.TemplateField(req),
	}
	pageData["Flashes"], err = h.flashes.GetAll(rw, req)
	if err != nil {
		return nil, err
	}

	return pageData, nil
}

func (h noticeHandler) save(rw http.ResponseWriter, req *http.Request) {
	ctx := req.Context()

	if req.Method != "POST" {
		err := weberrors.ErrBadRequest{Where: "http method type", Details: fmt.Errorf("add translation only accepts POST requests, sorry!")}
		h.r.Error(rw, req, http.StatusMethodNotAllowed, err)
		return
	}

	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "form data", Details: err}
		h.r.Error(rw, req, http.StatusInternalServerError, err)
		return
	}

	redirect := req.FormValue("redirect")
	if redirect == "" {
		noticesURL := h.urlTo(router.CompleteNoticeList)
		redirect = noticesURL.String()
	}

	// now, always redirect
	defer http.Redirect(rw, req, redirect, http.StatusSeeOther)

	if _, err := members.CheckAllowed(ctx, h.roomCfg, members.ActionChangeNotice); err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	var n roomdb.Notice
	n.ID, err = strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "id", Details: err}
		h.flashes.AddError(rw, req, err)
		return
	}

	n.Title = req.FormValue("title")
	if n.Title == "" {
		err = weberrors.ErrBadRequest{Where: "title", Details: fmt.Errorf("title can't be empty")}
		h.flashes.AddError(rw, req, err)
		return
	}

	// TODO: validate languages properly
	n.Language = req.FormValue("language")
	if n.Language == "" {
		err = weberrors.ErrBadRequest{Where: "language", Details: fmt.Errorf("language can't be empty")}
		h.flashes.AddError(rw, req, err)
		return
	}

	n.Content = req.FormValue("content")
	if n.Content == "" {
		err = weberrors.ErrBadRequest{Where: "content", Details: fmt.Errorf("content can't be empty")}
		h.flashes.AddError(rw, req, err)
		return
	}

	// https://github.com/russross/blackfriday/issues/575
	n.Content = strings.Replace(n.Content, "\r\n", "\n", -1)

	err = h.noticeDB.Save(req.Context(), &n)
	if err != nil {
		h.flashes.AddError(rw, req, err)
		return
	}

	h.flashes.AddMessage(rw, req, "NoticeUpdated")
}
