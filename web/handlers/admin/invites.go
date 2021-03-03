package admin

import (
	"fmt"
	"net/http"
	"strconv"

	"go.mindeco.de/http/render"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
	"github.com/vcraescu/go-paginator/v2/view"
)

type invitesH struct {
	r *render.Renderer

	db admindb.InviteService
}

func (h invitesH) overview(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	lst, err := h.db.List(req.Context())
	if err != nil {
		return nil, err
	}

	// Reverse the slice to provide recent-to-oldest results
	for i, j := 0, len(lst)-1; i < j; i, j = i+1, j-1 {
		lst[i], lst[j] = lst[j], lst[i]
	}

	// TODO: generalize paginator code

	count := len(lst)

	num, err := strconv.ParseInt(req.URL.Query().Get("page"), 10, 32)
	if err != nil {
		num = 1
	}
	page := int(num)
	if page < 1 {
		page = 1
	}

	paginator := paginator.New(adapter.NewSliceAdapter(lst), pageSize)
	paginator.SetPage(page)

	var entries admindb.ListEntries
	if err = paginator.Results(&entries); err != nil {
		return nil, fmt.Errorf("paginator failed with %w", err)
	}

	view := view.New(paginator)
	pagesSlice, err := view.Pages()
	if err != nil {
		return nil, fmt.Errorf("paginator view.Pages failed with %w", err)
	}
	if len(pagesSlice) == 0 {
		pagesSlice = []int{1}
	}
	last, err := view.Last()
	if err != nil {
		return nil, fmt.Errorf("paginator view.Last failed with %w", err)
	}
	firstInView := pagesSlice[0] == 1
	lastInView := false
	for _, num := range pagesSlice {
		if num == last {
			lastInView = true
		}
	}

	return map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(req),
		"Entries":        entries,
		"Count":          count,
		"Paginator":      paginator,
		"View":           view,
		"FirstInView":    firstInView,
		"LastInView":     lastInView,
	}, nil
}
