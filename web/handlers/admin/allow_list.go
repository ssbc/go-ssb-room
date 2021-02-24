package admin

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"

	"go.mindeco.de/http/render"
	refs "go.mindeco.de/ssb-refs"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
	"github.com/vcraescu/go-paginator/v2/view"
)

type allowListH struct {
	r *render.Renderer

	al admindb.AllowListService
}

const redirectTo = "/admin/members"

func (h allowListH) add(w http.ResponseWriter, req *http.Request) {
	if req.Method != "POST" {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request"))
		return
	}
	if err := req.ParseForm(); err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request: %w", err))
		return
	}

	newEntry := req.Form.Get("pub_key")
	newEntryParsed, err := refs.ParseFeedRef(newEntry)
	if err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request: %w", err))
		return
	}

	err = h.al.Add(req.Context(), *newEntryParsed)
	if err != nil {
		code := http.StatusInternalServerError
		var aa admindb.ErrAlreadyAdded
		if errors.As(err, &aa) {
			code = http.StatusBadRequest
			// TODO: localized error pages
			// h.r.Error(w, req, http.StatusBadRequest, weberrors.Localize())
			// return
		}

		h.r.Error(w, req, code, err)
		return
	}

	http.Redirect(w, req, redirectTo, http.StatusFound)
}

const pageSize = 20

func (h allowListH) overview(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	lst, err := h.al.List(req.Context())
	if err != nil {
		return nil, err
	}
	// Reverse the slice to provide recent-to-oldest results
	for i, j := 0, len(lst)-1; i < j; i, j = i+1, j-1 {
		lst[i], lst[j] = lst[j], lst[i]
	}
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
		panic(err)
	}

	view := view.New(paginator)
	pagesSlice, err := view.Pages()
	if err != nil {
		panic(err)
	}
	if len(pagesSlice) == 0 {
		pagesSlice = []int{1}
	}
	last, err := view.Last()
	if err != nil {
		panic(err)
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

// TODO: move to render package so that we can decide to not render a page during the controller
var ErrRedirected = errors.New("render: not rendered but redirected")

func (h allowListH) removeConfirm(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	id, err := strconv.ParseInt(req.URL.Query().Get("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		return nil, err
	}

	entry, err := h.al.GetByID(req.Context(), id)
	if err != nil {
		if errors.Is(err, admindb.ErrNotFound) {
			http.Redirect(rw, req, redirectTo, http.StatusFound)
			return nil, ErrRedirected
		}
		return nil, err
	}
	return map[string]interface{}{
		"Entry":          entry,
		csrf.TemplateTag: csrf.TemplateField(req),
	}, nil
}

func (h allowListH) remove(rw http.ResponseWriter, req *http.Request) {
	err := req.ParseForm()
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectTo, http.StatusFound)
		return
	}

	id, err := strconv.ParseInt(req.FormValue("id"), 10, 64)
	if err != nil {
		err = weberrors.ErrBadRequest{Where: "ID", Details: err}
		// TODO "flash" errors
		http.Redirect(rw, req, redirectTo, http.StatusFound)
		return
	}

	status := http.StatusFound
	err = h.al.RemoveID(req.Context(), id)
	if err != nil {
		if !errors.Is(err, admindb.ErrNotFound) {
			// TODO "flash" errors
			h.r.Error(rw, req, http.StatusInternalServerError, err)
			return
		}
		status = http.StatusNotFound
	}

	http.Redirect(rw, req, redirectTo, status)
}
