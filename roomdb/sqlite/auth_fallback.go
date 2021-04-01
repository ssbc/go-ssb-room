// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"golang.org/x/crypto/bcrypt"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.AuthFallbackService = (*AuthFallback)(nil)

type AuthFallback struct {
	db *sql.DB
}

// Check receives the username and password (in clear) and checks them accordingly.
// If it's a valid combination it returns the user ID, or an error if they are not.
func (af AuthFallback) Check(login, password string) (interface{}, error) {
	ctx := context.Background()
	found, err := models.FallbackPasswords(
		qm.Load("Member"),
		qm.Where("login = ?", login),
	).One(ctx, af.db)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return found, roomdb.ErrNotFound
		}
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(found.PasswordHash, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("auth/fallback: password missmatch")
	}

	return found.R.Member.ID, nil
}

func (af AuthFallback) Create(ctx context.Context, memberID int64, login string, password []byte) error {
	var newPasswordEntry models.FallbackPassword
	newPasswordEntry.MemberID = memberID
	newPasswordEntry.Login = login

	hashed, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth/fallback: failed to hash password for new user")
	}
	newPasswordEntry.PasswordHash = hashed

	err = newPasswordEntry.Insert(ctx, af.db, boil.Infer())
	if err != nil {
		return fmt.Errorf("auth/fallback: failed to insert new user: %w", err)
	}

	return nil
}
