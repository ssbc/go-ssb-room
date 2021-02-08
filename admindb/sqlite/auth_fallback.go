package sqlite

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/volatiletech/sqlboiler/v4/boil"

	"golang.org/x/crypto/bcrypt"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb/sqlite/models"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"

	"github.com/ssb-ngi-pointer/gossb-rooms/admindb"
)

// make sure to implement interfaces correctly
var _ admindb.AuthFallbackService = (*AuthFallback)(nil)

type AuthFallback struct {
	db *sql.DB
}

func (ah AuthFallback) Check(name, password string) (interface{}, error) {
	ctx := context.Background()
	found, err := models.AuthFallbacks(qm.Where("name = ?", name)).One(ctx, ah.db)
	if err != nil {
		return nil, err
	}

	err = bcrypt.CompareHashAndPassword(found.PasswordHash, []byte(password))
	if err != nil {
		return nil, fmt.Errorf("auth/fallback: password missmatch")
	}

	return found.ID, nil
}

func (ah AuthFallback) Create(name string, password []byte) error {
	var u models.AuthFallback
	u.Name = name

	hashed, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("auth/fallback: failed to hash password for new user")
	}

	u.PasswordHash = hashed

	ctx := context.Background()
	err = u.Insert(ctx, ah.db, boil.Infer())
	if err != nil {
		return fmt.Errorf("auth/fallback: failed to insert new user: %w", err)
	}

	return nil
}
