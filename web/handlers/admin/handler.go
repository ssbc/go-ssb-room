// SPDX-License-Identifier: MIT

// Package admin implements the dashboard for admins and moderators to change and control aspects of the room.
// Including aliases, allow/deny list managment, invites and settings of the room.
package admin

import (
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/vcraescu/go-paginator/v2"
	"github.com/vcraescu/go-paginator/v2/adapter"
	"github.com/vcraescu/go-paginator/v2/view"
	"go.mindeco.de/http/render"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomstate"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
)

// HTMLTemplates define the list of files the template system should load.
var HTMLTemplates = []string{
	"admin/dashboard.tmpl",
	"admin/menu.tmpl",

	"admin/aliases.tmpl",
	"admin/aliases-revoke-confirm.tmpl",

	"admin/denied-keys.tmpl",
	"admin/denied-keys-remove-confirm.tmpl",

	"admin/invite-list.tmpl",
	"admin/invite-revoke-confirm.tmpl",
	"admin/invite-created.tmpl",

	"admin/notice-edit.tmpl",

	"admin/members.tmpl",
	"admin/members-remove-confirm.tmpl",
}

// Databases is an option struct that encapsualtes the required database services
type Databases struct {
	Aliases       roomdb.AliasesService
	Config        roomdb.RoomConfig // cblgh: kind of confusing that we have two identically named structs, in different http handler contexts?
	DeniedKeys    roomdb.DeniedKeysService
	Invites       roomdb.InvitesService
	Notices       roomdb.NoticesService
	Members       roomdb.MembersService
	PinnedNotices roomdb.PinnedNoticesService
}

// Handler supplies the elevated access pages to known users.
// It is not registering on the mux router like other pages to clean up the authorize flow.
func Handler(
	domainName string,
	r *render.Renderer,
	roomState *roomstate.Manager,
	fh *weberrors.FlashHelper,
	dbs Databases,
) http.Handler {
	mux := &http.ServeMux{}

	mux.HandleFunc("/dashboard", r.HTML("admin/dashboard.tmpl", func(rw http.ResponseWriter, req *http.Request) (interface{}, error) {
		onlineRefs := roomState.List()
		onlineCount := len(onlineRefs)
		memberCount, err := dbs.Members.Count(req.Context())
		if err != nil {
			return nil, fmt.Errorf("failed to count members: %w", err)
		}
		inviteCount, err := dbs.Invites.Count(req.Context())
		if err != nil {
			return nil, fmt.Errorf("failed to count invites: %w", err)
		}
		deniedCount, err := dbs.DeniedKeys.Count(req.Context())
		if err != nil {
			return nil, fmt.Errorf("failed to count denied keys: %w", err)
		}
		return struct {
			OnlineRefs  []string
			OnlineCount int
			MemberCount uint
			InviteCount uint
			DeniedCount uint
		}{onlineRefs, onlineCount, memberCount, inviteCount, deniedCount}, nil
	}))
	mux.HandleFunc("/menu", r.HTML("admin/menu.tmpl", func(w http.ResponseWriter, req *http.Request) (interface{}, error) {
		return map[string]interface{}{}, nil
	}))

	var ah = aliasesHandler{
		r:       r,
		flashes: fh,

		db: dbs.Aliases,
	}
	mux.HandleFunc("/aliases", r.HTML("admin/aliases.tmpl", ah.overview))
	mux.HandleFunc("/aliases/revoke/confirm", r.HTML("admin/aliases-revoke-confirm.tmpl", ah.revokeConfirm))
	mux.HandleFunc("/aliases/revoke", ah.revoke)

	var dh = deniedKeysHandler{
		r:       r,
		flashes: fh,

		db: dbs.DeniedKeys,
	}
	mux.HandleFunc("/denied", r.HTML("admin/denied-keys.tmpl", dh.overview))
	mux.HandleFunc("/denied/add", dh.add)
	mux.HandleFunc("/denied/remove/confirm", r.HTML("admin/denied-keys-remove-confirm.tmpl", dh.removeConfirm))
	mux.HandleFunc("/denied/remove", dh.remove)

	var mh = membersHandler{
		r:       r,
		flashes: fh,

		db: dbs.Members,
	}
	mux.HandleFunc("/members", r.HTML("admin/members.tmpl", mh.overview))
	mux.HandleFunc("/members/add", mh.add)
	mux.HandleFunc("/members/change-role", mh.changeRole)
	mux.HandleFunc("/members/remove/confirm", r.HTML("admin/members-remove-confirm.tmpl", mh.removeConfirm))
	mux.HandleFunc("/members/remove", mh.remove)

	var ih = invitesHandler{
		r:       r,
		flashes: fh,

		db:     dbs.Invites,
		config: dbs.Config,

		domainName: domainName,
	}

	mux.HandleFunc("/invites", r.HTML("admin/invite-list.tmpl", ih.overview))
	mux.HandleFunc("/invites/create", r.HTML("admin/invite-created.tmpl", ih.create))
	mux.HandleFunc("/invites/revoke/confirm", r.HTML("admin/invite-revoke-confirm.tmpl", ih.revokeConfirm))
	mux.HandleFunc("/invites/revoke", ih.revoke)

	var nh = noticeHandler{
		r:       r,
		flashes: fh,

		noticeDB: dbs.Notices,
		pinnedDB: dbs.PinnedNotices,
	}
	mux.HandleFunc("/notice/edit", r.HTML("admin/notice-edit.tmpl", nh.edit))
	mux.HandleFunc("/notice/translation/draft", r.HTML("admin/notice-edit.tmpl", nh.draftTranslation))
	mux.HandleFunc("/notice/translation/add", nh.addTranslation)
	mux.HandleFunc("/notice/save", nh.save)

	// path:/ matches everything that isn't registerd (ie. its the "Not Found handler")
	mux.HandleFunc("/", http.HandlerFunc(func(rw http.ResponseWriter, req *http.Request) {
		r.Error(rw, req, 404, weberrors.ErrNotFound{What: req.URL.Path})
	}))

	return customStripPrefix("/admin", mux)
}

// how many elements does a paginated page have by default
const defaultPageSize = 20

// paginate receives the total slice and it's length/count, a URL query for the 'limit' and which 'page'.
//
// The members of the map are:
//	Entries: the paginated slice
//	Count: the total number of the whole, unpaginated list
//	FirstInView: a bool thats true if you render the first page
//	LastInView: a bool thats true if you render the last page
//	Paginator and View: helpers for rendering the page accessor (see github.com/vcraescu/go-paginator)
//
// TODO: we could return a struct instead but then need to re-think how we embedd it into all the pages where we need it.
//  Maybe renderData["Pages"] = paginatedData
func paginate(total interface{}, count int, qry url.Values) (map[string]interface{}, error) {
	pageSize, err := strconv.Atoi(qry.Get("limit"))
	if err != nil {
		pageSize = defaultPageSize
	}

	page, err := strconv.Atoi(qry.Get("page"))
	if err != nil || page < 1 {
		page = 1
	}

	paginator := paginator.New(adapter.NewSliceAdapter(total), pageSize)
	paginator.SetPage(page)

	var entries []interface{}
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

	return map[string]interface{}{
		"Entries":     entries,
		"Count":       count,
		"Paginator":   paginator,
		"View":        view,
		"FirstInView": pagesSlice[0] == 1,
		"LastInView":  pagesSlice[len(pagesSlice)-1] == last,
	}, nil
}

// trim prefix if exists (workaround for named router problem)
func customStripPrefix(prefix string, h http.Handler) http.Handler {
	if prefix == "" {
		return h
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := strings.TrimPrefix(r.URL.Path, prefix)
		rp := strings.TrimPrefix(r.URL.RawPath, prefix)
		if len(p) < len(r.URL.Path) && (r.URL.RawPath == "" || len(rp) < len(r.URL.RawPath)) {
			r2 := new(http.Request)
			*r2 = *r
			r2.URL = new(url.URL)
			*r2.URL = *r.URL
			r2.URL.Path = p
			r2.URL.RawPath = rp
			h.ServeHTTP(w, r2)
		} else {
			h.ServeHTTP(w, r)
		}
	})
}
