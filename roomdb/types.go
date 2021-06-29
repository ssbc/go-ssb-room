// SPDX-License-Identifier: MIT

package roomdb

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	refs "go.mindeco.de/ssb-refs"
)

// ErrNotFound is returned by the admin db if an object couldn't be found.
var ErrNotFound = errors.New("roomdb: object not found")

// Alias is how the roomdb stores an alias.
type Alias struct {
	ID int64

	Name string // or "alias string" as the docs call it

	Feed refs.FeedRef // the ssb identity that belongs to the user

	Signature []byte
}

type ErrAliasTaken struct {
	Name string
}

func (e ErrAliasTaken) Error() string {
	return fmt.Sprintf("alias (%q) is already taken", e.Name)
}

// Member holds all the information an internal user of the room has.
type Member struct {
	ID      int64
	Role    Role
	PubKey  refs.FeedRef
	Aliases []Alias
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=PrivacyMode

type PrivacyMode uint

func (pm PrivacyMode) IsValid() error {
	if pm == ModeUnknown || pm > ModeRestricted {
		return errors.New("No such privacy mode")
	}
	return nil
}

func ParsePrivacyMode(val string) PrivacyMode {
	switch val {
	case "ModeOpen":
		fallthrough
	case "open":
		return ModeOpen
	case "ModeCommunity":
		fallthrough
	case "community":
		return ModeCommunity
	case "ModeRestricted":
		fallthrough
	case "restricted":
		return ModeRestricted
	default:
		return ModeUnknown
	}
}

// PrivacyMode describes the access mode the room server is currently running under.
// ModeOpen allows anyone to create an room invite
// ModeCommunity restricts invite creation to pre-existing room members (i.e. "internal users")
// ModeRestricted only allows admins and moderators to create room invitations
const (
	ModeUnknown PrivacyMode = iota
	ModeOpen
	ModeCommunity
	ModeRestricted
)

var AllPrivacyModes = []PrivacyMode{ModeOpen, ModeCommunity, ModeRestricted}

// Implements the SQL marshaling interfaces (Scanner for Scan & Valuer for Value) for PrivacyMode

// Scan implements https://pkg.go.dev/database/sql#Scanner to read integers into a privacy mode
func (pm *PrivacyMode) Scan(src interface{}) error {
	dbValue, ok := src.(int64)
	if !ok {
		return fmt.Errorf("unexpected type: %T", src)
	}

	privacyMode := PrivacyMode(dbValue)

	err := privacyMode.IsValid()
	if err != nil {
		return err
	}

	*pm = privacyMode
	return nil
}

// Value returns privacy mode references as int64 to the database.
// https://pkg.go.dev/database/sql/driver#Valuer
func (pm PrivacyMode) Value() (driver.Value, error) {
	return driver.Value(int64(pm)), nil
}

//go:generate go run golang.org/x/tools/cmd/stringer -type=Role

// Role describes the authorization level of an internal user (or member).
// Valid roles are Member, Moderator or Admin.
// The zero value Uknown is used to detect missing initializion while not falling into a bad default.
type Role uint

func (r Role) IsValid() error {
	if r == RoleUnknown {
		return errors.New("unknown member role")
	}
	if r > RoleAdmin {
		return errors.New("invalid member role")
	}
	return nil
}

const (
	RoleUnknown Role = iota
	RoleMember
	RoleModerator
	RoleAdmin
)

var (
	roleAdminString  = RoleAdmin.String()
	roleModString    = RoleModerator.String()
	roleMemberString = RoleMember.String()
)

// UnmarshalText checks if a string is a valid role
func (r *Role) UnmarshalText(text []byte) error {
	roleStr := string(text)
	switch roleStr {

	case roleAdminString:
		*r = RoleAdmin

	case roleModString:
		*r = RoleModerator

	case roleMemberString:
		*r = RoleMember

	default:
		return fmt.Errorf("unknown member role: %q", roleStr)
	}

	return nil
}

type ErrAlreadyAdded struct {
	Ref refs.FeedRef
}

func (aa ErrAlreadyAdded) Error() string {
	return fmt.Sprintf("roomdb: the item (%s) is already on the list", aa.Ref.Ref())
}

// Invite is a combination of an invite id, who created it and when.
// The token itself is only visible from the db.Create function and stored hashed in the database
type Invite struct {
	ID int64

	CreatedBy Member
	CreatedAt time.Time
}

// ListEntry values are returned by the DenyListServices
type ListEntry struct {
	ID     int64
	PubKey refs.FeedRef

	CreatedAt time.Time
	Comment   string
}

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

func (n *PinnedNoticeName) UnmarshalJSON(input []byte) error {
	var str string
	if err := json.Unmarshal(input, &str); err != nil {
		return err
	}

	newNoticeName := PinnedNoticeName(str)

	if !newNoticeName.Valid() {
		return fmt.Errorf("PinnedNoticeName: invalid notice %q", str)
	}

	*n = newNoticeName
	return nil
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
