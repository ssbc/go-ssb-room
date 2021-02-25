package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/russross/blackfriday/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
)

type noticeHandler struct {
	pinned  admindb.PinnedNoticesService
	notices admindb.NoticesService
}

func (h noticeHandler) list(rw http.ResponseWriter, req *http.Request) (interface{}, error) {

	lst, err := h.pinned.List(req.Context())
	if err != nil {
		return nil, err
	}

	return struct {
		AllNotices admindb.SortedPinnedNotices
	}{lst.Sorted()}, nil
}

type noticeShowData struct {
	ID              int64
	Title, Language string
	Content         template.HTML
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

	return noticeShowData{
		ID:       noticeID,
		Title:    notice.Title,
		Content:  template.HTML(markdown),
		Language: notice.Language,
	}, nil
}
