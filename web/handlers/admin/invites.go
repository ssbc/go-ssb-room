package admin

import (
	"fmt"
	"net/http"

	"go.mindeco.de/http/render"

	"github.com/gorilla/csrf"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/web/user"
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

	pageData, err := paginate(lst, len(lst), req.URL.Query())
	if err != nil {
		return nil, err
	}

	pageData[csrf.TemplateTag] = csrf.TemplateField(req)

	return pageData, nil
}

func (h invitesH) create(w http.ResponseWriter, req *http.Request) {
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


	aliasSuggestion := req.Form.Get("alias_suggestion")

	token, err := h.db.Create(req.Context(), user.ID, aliasSuggestion)
	if err != nil {
		h.r.Error(w, req, http.StatusInternalServerError, err)
		return
	}

	fmt.Println("use me:", token)

	http.Redirect(w, req, "/admin/invites", http.StatusFound)
}
