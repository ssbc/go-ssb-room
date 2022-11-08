// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package handlers

import (
	"encoding/json"
	"go.mindeco.de/http/render"
	"html/template"
	"net/http"
	"strconv"

	"github.com/russross/blackfriday/v2"

	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/web/errors"
)

type noticeHandler struct {
	render  *render.Renderer
	flashes *errors.FlashHelper

	pinned  roomdb.PinnedNoticesService
	notices roomdb.NoticesService
}

type noticesListData struct {
	AllNotices roomdb.SortedPinnedNotices
	Flashes    []errors.FlashMessage
}

func (h noticeHandler) list(rw http.ResponseWriter, req *http.Request) {
	var responder listNoticesResponder
	switch req.URL.Query().Get("encoding") {
	case "json":
		responder = newListNoticesJSONResponder(rw)
	default:
		responder = newListNoticesHTMLResponder(h.render, rw, req)
	}

	lst, err := h.pinned.List(req.Context())
	if err != nil {
		responder.RenderError(err)
		return
	}

	flashes, err := h.flashes.GetAll(rw, req)
	if err != nil {
		responder.RenderError(err)
		return
	}

	pageData := noticesListData{
		AllNotices: lst.Sorted(),
		Flashes:    flashes,
	}

	responder.Render(pageData)
}

type noticeShowData struct {
	ID              int64
	Title, Language string
	Content         template.HTML

	Flashes []errors.FlashMessage
}

func (h noticeHandler) show(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	noticeID, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		return nil, errors.ErrBadRequest{Where: "notice ID", Details: err}
	}

	notice, err := h.notices.GetByID(req.Context(), noticeID)
	if err != nil {
		return nil, err
	}

	markdown := blackfriday.Run([]byte(notice.Content), blackfriday.WithNoExtensions())

	pageData := noticeShowData{
		ID:       noticeID,
		Title:    notice.Title,
		Content:  template.HTML(markdown),
		Language: notice.Language,
	}

	pageData.Flashes, err = h.flashes.GetAll(rw, req)
	if err != nil {
		return nil, err
	}

	return pageData, nil
}

type listNoticesResponder interface {
	Render(noticesListData)
	RenderError(error)
}

type listNoticesJSONResponder struct {
	rw http.ResponseWriter
}

func newListNoticesJSONResponder(rw http.ResponseWriter) *listNoticesJSONResponder {
	return &listNoticesJSONResponder{rw: rw}
}

func (l listNoticesJSONResponder) Render(data noticesListData) {
	l.rw.Header().Set("Content-Type", "application/json")
	var pinnedNotices []listNoticesJSONResponsePinnedNotice
	for _, pinnedNotice := range data.AllNotices {
		v := listNoticesJSONResponsePinnedNotice{
			Name:    string(pinnedNotice.Name),
			Notices: nil,
		}
		for _, notice := range pinnedNotice.Notices {
			v.Notices = append(v.Notices, listNoticesJSONResponseNotice{
				ID:       notice.ID,
				Title:    notice.Title,
				Content:  notice.Content,
				Language: notice.Language,
			})
		}
		pinnedNotices = append(pinnedNotices, v)
	}

	var resp = listNoticesJSONResponse{
		PinnedNotices: pinnedNotices,
	}
	json.NewEncoder(l.rw).Encode(resp)
}

func (l listNoticesJSONResponder) RenderError(err error) {
	l.rw.Header().Set("Content-Type", "application/json")
	l.rw.WriteHeader(http.StatusInternalServerError)
	json.NewEncoder(l.rw).Encode(struct {
		Status string `json:"status"`
		Error  string `json:"error"`
	}{"error", err.Error()})
}

type listNoticesHTMLResponder struct {
	renderer *render.Renderer
	rw       http.ResponseWriter
	req      *http.Request
}

func newListNoticesHTMLResponder(renderer *render.Renderer, rw http.ResponseWriter, req *http.Request) *listNoticesHTMLResponder {
	return &listNoticesHTMLResponder{renderer: renderer, rw: rw, req: req}
}

func (l listNoticesHTMLResponder) Render(data noticesListData) {
	l.renderer.HTML("notice/list.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		return data, nil
	})(l.rw, l.req)
}

func (l listNoticesHTMLResponder) RenderError(err error) {
	l.renderer.Error(l.rw, l.req, http.StatusInternalServerError, err)
}

type listNoticesJSONResponse struct {
	PinnedNotices []listNoticesJSONResponsePinnedNotice `json:"pinned_notices"`
}

type listNoticesJSONResponsePinnedNotice struct {
	Name    string                          `json:"name"`
	Notices []listNoticesJSONResponseNotice `json:"notices"`
}

type listNoticesJSONResponseNotice struct {
	ID       int64  `json:"id"`
	Title    string `json:"title"`
	Content  string `json:"content"`
	Language string `json:"language"`
}
