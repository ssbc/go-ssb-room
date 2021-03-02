// SPDX-License-Identifier: MIT

package admindb

import (
	"database/sql/driver"
	"errors"
	"fmt"
	"sort"

	refs "go.mindeco.de/ssb-refs"
)

// ErrNotFound is returned by the admin db if an object couldn't be found.
var ErrNotFound = errors.New("admindb: object not found")

// User holds all the information an authenticated user of the site has.
type User struct {
	ID   int64
	Name string
}

type ErrAlreadyAdded struct {
	Ref refs.FeedRef
}

func (aa ErrAlreadyAdded) Error() string {
	return fmt.Sprintf("admindb: the item (%s) is already on the list", aa.Ref.Ref())
}

// Invite is a combination of an invite id, who created it and an (optional) alias suggestion.
// The token itself is only visible from the db.Create function and stored hashed in the database
type Invite struct {
	ID int64

	CreatedBy User

	AliasSuggestion string
}

// ListEntry values are returned by Allow- and DenyListServices
type ListEntry struct {
	ID     int64
	PubKey refs.FeedRef
}

// ListEntries is a slice of ListEntries
type ListEntries []ListEntry

// DBFeedRef wraps a feed reference and implements the SQL marshaling interfaces.
type DBFeedRef struct{ refs.FeedRef }

// Scan implements https://pkg.go.dev/database/sql#Scanner to read strings into feed references.
func (r *DBFeedRef) Scan(src interface{}) error {
	str, ok := src.(string)
	if !ok {
		return fmt.Errorf("unexpected type: %T", src)
	}

	fr, err := refs.ParseFeedRef(str)
	if err != nil {
		return err
	}

	r.FeedRef = *fr
	return nil
}

// Value returns feed references as strings to the database.
// https://pkg.go.dev/database/sql/driver#Valuer
func (r DBFeedRef) Value() (driver.Value, error) {
	return driver.Value(r.Ref()), nil
}

// PinnedNoticeName holds a name of a well known part of the page with a fixed location.
// These also double as the i18n labels.
type PinnedNoticeName string

func (n PinnedNoticeName) String() string {
	return string(n)
}

// These are the well known names that the room page will display
const (
	NoticeDescription   PinnedNoticeName = "NoticeDescription"
	NoticeNews          PinnedNoticeName = "NoticeNews"
	NoticePrivacyPolicy PinnedNoticeName = "NoticePrivacyPolicy"
	NoticeCodeOfConduct PinnedNoticeName = "NoticeCodeOfConduct"
)

// Valid returns true if the page name is well known.
func (n PinnedNoticeName) Valid() bool {
	return n == NoticeNews ||
		n == NoticeDescription ||
		n == NoticePrivacyPolicy ||
		n == NoticeCodeOfConduct
}

type PinnedNotices map[PinnedNoticeName][]Notice

// Notice holds the title and content of a page that is user generated
type Notice struct {
	ID       int64
	Title    string
	Content  string
	Language string
}

type PinnedNotice struct {
	Name    PinnedNoticeName
	Notices []Notice
}

type SortedPinnedNotices []PinnedNotice

// Sorted returns a sorted list of the map, by the key names
func (pn PinnedNotices) Sorted() SortedPinnedNotices {

	lst := make(SortedPinnedNotices, 0, len(pn))

	for name, notices := range pn {
		lst = append(lst, PinnedNotice{
			Name:    name,
			Notices: notices,
		})
	}

	sort.Sort(lst)
	return lst
}

var _ sort.Interface = (SortedPinnedNotices)(nil)

func (byName SortedPinnedNotices) Len() int { return len(byName) }

func (byName SortedPinnedNotices) Less(i, j int) bool {
	return byName[i].Name < byName[j].Name
}

func (byName SortedPinnedNotices) Swap(i, j int) {
	byName[i], byName[j] = byName[j], byName[i]
}
