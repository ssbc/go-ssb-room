// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/volatiletech/sqlboiler/v4/boil"

	"golang.org/x/crypto/bcrypt"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb/sqlite/models"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/go-ssb-room/roomdb"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.AuthFallbackService = (*AuthFallback)(nil)

type AuthFallback struct {
	db *sql.DB
}

// Check receives the username and password (in clear) and checks them accordingly.
// If it's a valid combination it returns the user ID, or an error if they are not.
func (af AuthFallback) Check(name, password string) (interface{}, error) {
	ctx := context.Background()
	found, err := models.AuthFallbacks(qm.Where("name = ?", name)).One(ctx, af.db)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(found.PasswordHash, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("auth/fallback: password missmatch")
	}

	return found.ID, nil
}

func (af AuthFallback) Create(ctx context.Context, name string, password []byte) (int64, error) {
	var u models.AuthFallback
	u.Name = name

	hashed, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return -1, fmt.Errorf("auth/fallback: failed to hash password for new user")
	}

	u.PasswordHash = hashed

	err = u.Insert(ctx, af.db, boil.Infer())
	if err != nil {
		return -1, fmt.Errorf("auth/fallback: failed to insert new user: %w", err)
	}

	return u.ID, nil
}

func (af AuthFallback) GetByID(ctx context.Context, uid int64) (*roomdb.User, error) {
	modelU, err := models.FindAuthFallback(ctx, af.db, uid)
	if err != nil {
		return nil, err
	}
	return &roomdb.User{
		ID:   modelU.ID,
		Name: modelU.Name,
	}, nil
}
