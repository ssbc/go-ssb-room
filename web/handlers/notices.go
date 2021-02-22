package handlers

import (
	"html/template"
	"net/http"
	"strconv"

	"github.com/russross/blackfriday/v2"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
)

type noticeRenderer struct {
	notices admindb.NoticesService
}

type noticeData struct {
	ID              int64
	Title, Language string
	Content         template.HTML
}

func (pr noticeRenderer) render(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	noticeID, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		return nil, errors.ErrBadRequest{Where: "notice ID", Details: err}
	}

	notice, err := pr.notices.GetByID(req.Context(), noticeID)
	if err != nil {
		return nil, err
	}

	markdown := blackfriday.Run([]byte(notice.Content), blackfriday.WithNoExtensions())

	return noticeData{
		ID:       noticeID,
		Title:    notice.Title,
		Content:  template.HTML(markdown),
		Language: notice.Language,
	}, nil
}
