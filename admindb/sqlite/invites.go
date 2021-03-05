package sqlite

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"fmt"

	"github.com/friendsofgo/errors"
	"github.com/mattn/go-sqlite3"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/go-ssb-room/admindb"
	"github.com/ssb-ngi-pointer/go-ssb-room/admindb/sqlite/models"
	refs "go.mindeco.de/ssb-refs"
)

// compiler assertion to ensure the struct fullfills the interface
var _ admindb.InviteService = (*Invites)(nil)

// Invites implements the admindb.InviteService.
// Tokens are stored as sha256 hashes on disk to protect against attackers gaining database read-access.
type Invites struct {
	db *sql.DB

	allowList *AllowList
}

// Create creates a new invite for a new member. It returns the token or an error.
// createdBy is user ID of the admin or moderator who created it.
// aliasSuggestion is optional (empty string is fine) but can be used to disambiguate open invites. (See https://github.com/ssb-ngi-pointer/rooms2/issues/21)
// The returned token is base64 URL encoded and has tokenLength when decoded.
func (i Invites) Create(ctx context.Context, createdBy int64, aliasSuggestion string) (string, error) {
	var newInvite = models.Invite{
		CreatedBy:       createdBy,
		AliasSuggestion: aliasSuggestion,
	}

	tokenBytes := make([]byte, tokenLength)

	err := transact(i.db, func(tx *sql.Tx) error {

		inserted := false
	trying:
		for tries := 100; tries > 0; tries-- {
			// generate an invite code
			rand.Read(tokenBytes)

			// hash the binary of the token for storage
			h := sha256.New()
			h.Write(tokenBytes)
			newInvite.Token = fmt.Sprintf("%x", h.Sum(nil))

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
// Tokens need to be base64 URL encoded and when decoded be of tokenLength.
func (i Invites) Consume(ctx context.Context, token string, newMember refs.FeedRef) (admindb.Invite, error) {
	var inv admindb.Invite

	hashedToken, err := getHashedToken(token)
	if err != nil {
		return inv, err
	}

	err = transact(i.db, func(tx *sql.Tx) error {
		entry, err := models.Invites(
			qm.Where("active = true AND token = ?", hashedToken),
			qm.Load("CreatedByAuthFallback"),
		).One(ctx, tx)
		if err != nil {
			return err
		}

		err = i.allowList.add(ctx, tx, newMember)
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
		inv.AliasSuggestion = entry.AliasSuggestion
		inv.CreatedBy.ID = entry.R.CreatedByAuthFallback.ID
		inv.CreatedBy.Name = entry.R.CreatedByAuthFallback.Name

		return nil
	})

	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inv, admindb.ErrNotFound
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

func (i Invites) GetByToken(ctx context.Context, token string) (admindb.Invite, error) {
	var inv admindb.Invite

	ht, err := getHashedToken(token)
	if err != nil {
		return inv, err
	}

	entry, err := models.Invites(
		qm.Where("active = true AND token = ?", ht),
		qm.Load("CreatedByAuthFallback"),
	).One(ctx, i.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inv, admindb.ErrNotFound
		}
		return inv, err
	}

	inv.ID = entry.ID
	inv.AliasSuggestion = entry.AliasSuggestion
	inv.CreatedBy.ID = entry.R.CreatedByAuthFallback.ID
	inv.CreatedBy.Name = entry.R.CreatedByAuthFallback.Name

	return inv, nil
}

func (i Invites) GetByID(ctx context.Context, id int64) (admindb.Invite, error) {
	var inv admindb.Invite

	entry, err := models.Invites(
		qm.Where("active = true AND id = ?", id),
		qm.Load("CreatedByAuthFallback"),
	).One(ctx, i.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return inv, admindb.ErrNotFound
		}
		return inv, err
	}

	inv.ID = entry.ID
	inv.AliasSuggestion = entry.AliasSuggestion
	inv.CreatedBy.ID = entry.R.CreatedByAuthFallback.ID
	inv.CreatedBy.Name = entry.R.CreatedByAuthFallback.Name

	return inv, nil
}

// List returns a list of all the valid invites
func (i Invites) List(ctx context.Context) ([]admindb.Invite, error) {
	var invs []admindb.Invite

	err := transact(i.db, func(tx *sql.Tx) error {
		entries, err := models.Invites(
			qm.Where("active = true"),
			qm.Load("CreatedByAuthFallback"),
		).All(ctx, tx)
		if err != nil {
			return err
		}

		invs = make([]admindb.Invite, len(entries))
		for i, e := range entries {
			var inv admindb.Invite
			inv.ID = e.ID
			inv.AliasSuggestion = e.AliasSuggestion
			inv.CreatedBy.ID = e.R.CreatedByAuthFallback.ID
			inv.CreatedBy.Name = e.R.CreatedByAuthFallback.Name

			invs[i] = inv
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return invs, nil
}

// Revoke removes a active invite and invalidates it for future use.
func (i Invites) Revoke(ctx context.Context, id int64) error {
	return transact(i.db, func(tx *sql.Tx) error {
		entry, err := models.Invites(
			qm.Where("active = true AND id = ?", id),
		).One(ctx, tx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return admindb.ErrNotFound
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

const tokenLength = 50

func getHashedToken(b64tok string) (string, error) {
	tokenBytes, err := base64.URLEncoding.DecodeString(b64tok)
	if err != nil {
		return "", err
	}

	if n := len(tokenBytes); n != tokenLength {
		return "", fmt.Errorf("admindb: invalid invite token length (only got %d bytes)", n)
	}

	// hash the binary of the passed token
	h := sha256.New()
	h.Write(tokenBytes)
	return fmt.Sprintf("%x", h.Sum(nil)), nil
}
