package sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/friendsofgo/errors"
	"github.com/mattn/go-sqlite3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
	refs "go.mindeco.de/ssb-refs"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.InvitesService = (*Invites)(nil)

// Invites implements the roomdb.InviteService.
// Tokens are stored as sha256 hashes on disk to protect against attackers gaining database read-access.
type Invites struct {
	db *sql.DB

	members Members
}

// Create creates a new invite for a new member. It returns the token or an error.
// createdBy is user ID of the admin or moderator who created it.
// aliasSuggestion is optional (empty string is fine) but can be used to disambiguate open invites. (See https://github.com/ssb-ngi-pointer/rooms2/issues/21)
// The returned token is base64 URL encoded and has inviteTokenLength when decoded.
func (i Invites) Create(ctx context.Context, createdBy int64) (string, error) {
	var newInvite = models.Invite{
		CreatedBy: createdBy,
	}

	tokenBytes := make([]byte, inviteTokenLength)

	err := transact(i.db, func(tx *sql.Tx) error {

		inserted := false
	trying:
		for tries := 100; tries > 0; tries-- {
			// generate an invite code
			rand.Read(tokenBytes)

			// see comment on migrations/6-invite-createdAt.sql
			newInvite.CreatedAt = time.Now()

			// hash the binary of the token for storage
			h := sha256.New()
			h.Write(tokenBytes)
			newInvite.HashedToken = fmt.Sprintf("%x", h.Sum(nil))

			// insert the new invite
			err := newInvite.Insert(ctx, tx, boil.Infer())
			if err != nil {
				var sqlErr sqlite3.Error
				if errors.As(err, &sqlErr) && sqlErr.ExtendedCode == sqlite3.ErrConstraintUnique {
					// generated an existing token, retry
					continue trying
				}
				return err
			}
			inserted = true
			break // no error means it worked!
		}

		if !inserted {
			return errors.New("admindb: failed to generate an invite token in a reasonable amount of time")
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(tokenBytes), nil
}

// Consume checks if the passed token is still valid. If it is it adds newMember to the members of the room and invalidates the token.
// If the token isn't valid, it returns an error.
// Tokens need to be base64 URL encoded and when decoded be of inviteTokenLength.
func (i Invites) Consume(ctx context.Context, token string, newMember refs.FeedRef) (roomdb.Invite, error) {
	var inv roomdb.Invite

	hashedToken, err := getHashedToken(token)
	if err != nil {
		return inv, err
	}

	err = transact(i.db, func(tx *sql.Tx) error {
		entry, err := models.Invites(
			qm.Where("active = true AND hashed_token = ?", hashedToken),
			qm.Load("CreatedByMember"),
		).One(ctx, tx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return roomdb.ErrNotFound
			}
			return err
		}

		_, err = i.members.add(ctx, tx, newMember, roomdb.RoleMember)
		if err != nil {
			return err
		}

		// invalidate the invite for consumption
		entry.Active = false
		_, err = entry.Update(ctx, tx, boil.Whitelist("active"))
		if err != nil {
			return err
		}

		inv.ID = entry.ID
		inv.CreatedAt = entry.CreatedAt
		inv.CreatedBy.ID = entry.R.CreatedByMember.ID
		inv.CreatedBy.Role = roomdb.Role(entry.R.CreatedByMember.Role)

		return nil
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inv, roomdb.ErrNotFound
		}
		return inv, err
	}

	return inv, nil
}

// since invites are marked as inavalid so that the code can't be generated twice,
// they need to be deleted periodically.
func deleteConsumedInvites(tx boil.ContextExecutor) error {
	_, err := models.Invites(qm.Where("active = false")).DeleteAll(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("admindb: failed to delete used invites: %w", err)
	}
	return nil
}

func (i Invites) GetByToken(ctx context.Context, token string) (roomdb.Invite, error) {
	var inv roomdb.Invite

	ht, err := getHashedToken(token)
	if err != nil {
		return inv, err
	}

	entry, err := models.Invites(
		qm.Where("active = true AND hashed_token = ?", ht),
		qm.Load("CreatedByMember"),
	).One(ctx, i.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inv, roomdb.ErrNotFound
		}
		return inv, err
	}

	inv.ID = entry.ID
	inv.CreatedAt = entry.CreatedAt
	inv.CreatedBy.ID = entry.R.CreatedByMember.ID
	inv.CreatedBy.Role = roomdb.Role(entry.R.CreatedByMember.Role)

	return inv, nil
}

func (i Invites) GetByID(ctx context.Context, id int64) (roomdb.Invite, error) {
	var inv roomdb.Invite

	entry, err := models.Invites(
		qm.Where("active = true AND id = ?", id),
		qm.Load("CreatedByMember"),
	).One(ctx, i.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inv, roomdb.ErrNotFound
		}
		return inv, err
	}

	inv.ID = entry.ID
	inv.CreatedAt = entry.CreatedAt
	inv.CreatedBy.ID = entry.R.CreatedByMember.ID
	inv.CreatedBy.Role = roomdb.Role(entry.R.CreatedByMember.Role)
	inv.CreatedBy.PubKey = entry.R.CreatedByMember.PubKey.FeedRef
	inv.CreatedBy.Aliases = i.members.getAliases(entry.R.CreatedByMember)

	return inv, nil
}

// List returns a list of all the valid invites
func (i Invites) List(ctx context.Context) ([]roomdb.Invite, error) {
	var invs []roomdb.Invite

	err := transact(i.db, func(tx *sql.Tx) error {
		entries, err := models.Invites(
			qm.Where("active = true"),
			qm.Load("CreatedByMember"),
		).All(ctx, tx)
		if err != nil {
			return err
		}

		invs = make([]roomdb.Invite, len(entries))
		for idx, e := range entries {
			var inv roomdb.Invite
			inv.ID = e.ID
			inv.CreatedAt = e.CreatedAt
			inv.CreatedBy.ID = e.R.CreatedByMember.ID
			inv.CreatedBy.PubKey = e.R.CreatedByMember.PubKey.FeedRef
			inv.CreatedBy.Aliases = i.members.getAliases(e.R.CreatedByMember)

			invs[idx] = inv
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return invs, nil
}

func (i Invites) Count(ctx context.Context) (uint, error) {
	count, err := models.Invites().Count(ctx, i.db)
	if err != nil {
		return 0, err
	}
	return uint(count), nil
}

// Revoke removes a active invite and invalidates it for future use.
func (i Invites) Revoke(ctx context.Context, id int64) error {
	return transact(i.db, func(tx *sql.Tx) error {
		entry, err := models.Invites(
			qm.Where("active = true AND id = ?", id),
		).One(ctx, tx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return roomdb.ErrNotFound
			}
			return err
		}

		entry.Active = false
		_, err = entry.Update(ctx, tx, boil.Whitelist("active"))
		if err != nil {
			return err
		}

		return nil
	})
}

const inviteTokenLength = 50

func getHashedToken(b64tok string) (string, error) {
	tokenBytes, err := base64.URLEncoding.DecodeString(b64tok)
	if err != nil {
		return "", err
	}

	if n := len(tokenBytes); n != inviteTokenLength {
		return "", fmt.Errorf("admindb: invalid invite token length (only got %d bytes)", n)
	}

	// hash the binary of the passed token
	h := sha256.New()
	h.Write(tokenBytes)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
