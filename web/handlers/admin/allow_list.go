package admin

import (
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"
	refs "go.mindeco.de/ssb-refs"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
)

type allowListH struct {
	r *render.Renderer

	al admindb.AllowListService
}

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
		fmt.Println("failed to parse: ", err)
		// TODO: proper error type
		h.r.Error(w, req, http.StatusBadRequest, fmt.Errorf("bad request: %w", err))
		return
	}

	err = h.al.Add(req.Context(), *newEntryParsed)
	if err != nil {
		// TODO: proper error type
		h.r.Error(w, req, http.StatusInternalServerError, fmt.Errorf("maybe exists?: %w", err))
		return
	}

	http.Redirect(w, req, "/admin/allow-list", http.StatusFound)
}

func (h allowListH) overview(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
	lst, err := h.al.List(req.Context())
	if err != nil {
		return nil, err
	}

	return struct {
		Entries []refs.FeedRef
		Count   int
	}{lst, len(lst)}, nil
}
