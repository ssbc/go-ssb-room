// SPDX-License-Identifier: MIT

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
	"go.mindeco.de/http/auth"
	"golang.org/x/crypto/bcrypt"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
	weberrors "github.com/ssb-ngi-pointer/go-ssb-room/web/errors"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.AuthFallbackService = (*AuthFallback)(nil)

type AuthFallback struct {
	db *sql.DB
}

var redirectPasswordAuthErr = weberrors.ErrRedirect{
	Path:   "/fallback/login",
	Reason: auth.ErrBadLogin,
}

// Check receives the loging and password (in clear) and checks them accordingly.
// Login might be a registered alias or a ssb id who belongs to a member.
// If it's a valid combination it returns the user ID, or an error if they are not.
func (af AuthFallback) Check(login, password string) (interface{}, error) {
	ctx := context.Background()

	var memberID = int64(-1)

	// try looking up an alias first
	loginIsAlias, err := models.Aliases(qm.Where("name = ?", login)).One(ctx, af.db)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			// something else went wrong
			return nil, err
		}

		// did not find an alias, try as ssb reference
		member, err := models.Members(qm.Where("pub_key = ?", login)).One(ctx, af.db)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, redirectPasswordAuthErr
		}

		// member found by ssb id, use their id
		memberID = member.ID
	} else {
		// found an alias, use the corresponding member ID
		memberID = loginIsAlias.MemberID
	}

	foundPassword, err := models.FallbackPasswords(qm.Where("member_id = ?", memberID)).One(ctx, af.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, redirectPasswordAuthErr
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(foundPassword.PasswordHash, []byte(password))
	if err != nil {
		return nil, redirectPasswordAuthErr
	}

	return foundPassword.MemberID, nil
}

func (af AuthFallback) SetPassword(ctx context.Context, memberID int64, password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth/fallback: failed to hash password for member")
	}

	// this is a silly upsert construction which sqlboiler-sqlite doesnt support nativly
	return transact(af.db, func(tx *sql.Tx) error {

		foundPassword, err := models.FallbackPasswords(qm.Where("member_id = ?", memberID)).One(ctx, tx)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				// something else went wrong
				return err
			}
			// not found => insert new entry
			var newPasswordEntry models.FallbackPassword
			newPasswordEntry.PasswordHash = hashed
			newPasswordEntry.MemberID = memberID
			err = newPasswordEntry.Insert(ctx, tx, boil.Infer())
			if err != nil {
				return fmt.Errorf("auth/fallback: failed to insert new user: %w", err)
			}
		} else {
			// found => update the entry
			foundPassword.PasswordHash = hashed
			_, err = foundPassword.Update(ctx, tx, boil.Whitelist("password_hash"))
			if err != nil {
				return fmt.Errorf("auth/fallback: failed to update password for member: %w", err)
			}
		}

		return nil
	})
}

func (af AuthFallback) SetPasswordWithToken(ctx context.Context, resetToken string, password string) error {
	hashed, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth/fallback: failed to hash password for member")
	}

	hashedTok, err := getHashedToken(resetToken)
	if err != nil {
		return fmt.Errorf("invalid password reset token")
	}

	// this is a silly upsert construction which sqlboiler-sqlite doesnt support nativly
	return transact(af.db, func(tx *sql.Tx) error {
		//  make sure its a valid one and load it
		resetEntry, err := models.FallbackResetTokens(qm.Where("active = true and hashed_token = ?", hashedTok)).One(ctx, tx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return fmt.Errorf("could not find the reset-token")
			}
			return err
		}

		// see that there is a password entry for the member in the reset entry
		foundPassword, err := models.FallbackPasswords(qm.Where("member_id = ?", resetEntry.ForMember)).One(ctx, tx)
		if err != nil {
			if !errors.Is(err, sql.ErrNoRows) {
				return err
			}
			// not found => insert new entry
			var newPasswordEntry models.FallbackPassword
			newPasswordEntry.PasswordHash = hashed
			newPasswordEntry.MemberID = resetEntry.ForMember
			err = newPasswordEntry.Insert(ctx, tx, boil.Infer())
			if err != nil {
				return fmt.Errorf("auth/fallback: failed to insert new fallback password for member: %w", err)
			}
		} else {
			// found it => update the entry
			foundPassword.PasswordHash = hashed
			_, err = foundPassword.Update(ctx, tx, boil.Whitelist("password_hash"))
			if err != nil {
				return fmt.Errorf("auth/fallback: failed to update password for member: %w", err)
			}
		}

		// finally, invalidate the token
		resetEntry.Active = false
		_, err = resetEntry.Update(ctx, tx, boil.Whitelist("active"))
		if err != nil {
			return fmt.Errorf("auth/fallback: failed to invalidate the reset entry: %w", err)
		}

		return nil
	})
}

func (af AuthFallback) CreateResetToken(ctx context.Context, createdByMember, forMember int64) (string, error) {
	var newResetToken = models.FallbackResetToken{
		CreatedBy: createdByMember,
		ForMember: forMember,
	}

	tokenBytes := make([]byte, inviteTokenLength)

	err := transact(af.db, func(tx *sql.Tx) error {

		inserted := false
	trying:
		for tries := 100; tries > 0; tries-- {
			// generate an invite code
			rand.Read(tokenBytes)

			// hash the binary of the token for storage
			h := sha256.New()
			h.Write(tokenBytes)
			newResetToken.HashedToken = fmt.Sprintf("%x", h.Sum(nil))

			// insert the new invite
			err := newResetToken.Insert(ctx, tx, boil.Infer())
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

// since reset tokens are marked as inavalid so that the code can't be generated twice,
// they need to be deleted periodically.
func deleteConsumedResetTokens(tx boil.ContextExecutor) error {
	_, err := models.FallbackResetTokens(qm.Where("active = false")).DeleteAll(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("admindb: failed to delete used reset tokens: %w", err)
	}
	return nil
}
