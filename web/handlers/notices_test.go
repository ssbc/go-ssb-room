package handlers

import (
	"net/http"
	"testing"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/stretchr/testify/assert"
)

// TestNoticeSmokeTest ensures the most basic notice serving is working
func TestNoticeSmokeTest(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	noticeData := admindb.Notice{
		ID:    1,
		Title: "Welcome!",
	}

	ts.NoticeDB.GetByIDReturns(noticeData, nil)

	html, res := ts.Client.GetHTML("/notice/show?id=1")
	a.Equal(http.StatusOK, res.Code, "wrong HTTP status code")
	a.Equal("Welcome!", html.Find("title").Text())
}

func TestNoticeMarkdownServedCorrectly(t *testing.T) {
	ts := setup(t)
	a := assert.New(t)

	markdown := `
Hello world!

## The loveliest of rooms is here
`
	noticeData := admindb.Notice{
		ID:      1,
		Title:   "Welcome!",
		Content: markdown,
	}

	ts.NoticeDB.GetByIDReturns(noticeData, nil)

	html, res := ts.Client.GetHTML("/notice/show?id=1")
	a.Equal(http.StatusOK, res.Code, "wrong HTTP status code")
	a.Equal("Welcome!", html.Find("title").Text())
	a.Equal("The loveliest of rooms is here", html.Find("h2").Text())
}
