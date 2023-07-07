// SPDX-FileCopyrightText: 2021 The NGI Pointer Secure-Scuttlebutt Team of 2020/2021
//
// SPDX-License-Identifier: MIT

package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/friendsofgo/errors"
	"github.com/volatiletech/sqlboiler/v4/boil"
	"github.com/volatiletech/sqlboiler/v4/queries/qm"
	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"

	"github.com/ssbc/go-ssb-room/v2/internal/randutil"
	"github.com/ssbc/go-ssb-room/v2/roomdb"
	"github.com/ssbc/go-ssb-room/v2/roomdb/sqlite/models"
)

// compiler assertion to ensure the struct fullfills the interface
var _ roomdb.AuthWithSSBService = (*AuthWithSSB)(nil)

type AuthWithSSB struct {
	db *sql.DB
}

const siwssbTokenLength = 32

// CreateToken is used to generate a token that is stored inside a cookie.
// It is used after a valid solution for a challenge was provided.
func (a AuthWithSSB) CreateToken(ctx context.Context, memberID int64) (string, error) {

	var newToken = models.SIWSSBSession{
		MemberID: memberID,
	}

	err := transact(a.db, func(tx *sql.Tx) error {

		// check the member is registerd
		if _, err := models.FindMember(ctx, tx, newToken.MemberID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return roomdb.ErrNotFound
			}
			return err
		}

		inserted := false
	trying: // keep trying until we inserted in unused token
		for tries := 100; tries > 0; tries-- {

			// generate an new token
			newToken.Token = randutil.String(siwssbTokenLength)

			// insert the new token
			cols := boil.Whitelist(models.SIWSSBSessionColumns.Token, models.SIWSSBSessionColumns.MemberID)
			err := newToken.Insert(ctx, tx, cols)
			if err != nil {
				var sqlErr *sqlite.Error
				if errors.As(err, &sqlErr) && sqlErr.Code() == sqlite3.SQLITE_CONSTRAINT_UNIQUE {
					// generated an existing token, retry
					continue trying
				}
				return err
			}
			inserted = true
			break // no error means it worked!
		}

		if !inserted {
			return errors.New("admindb: failed to generate a fresh token in a reasonable amount of time")
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return newToken.Token, nil
}

const sessionTimeout = time.Hour * 24

// CheckToken checks if the passed token is still valid and returns the member id if so
func (a AuthWithSSB) CheckToken(ctx context.Context, token string) (int64, error) {
	var memberID int64

	err := transact(a.db, func(tx *sql.Tx) error {
		session, err := models.SIWSSBSessions(qm.Where("token = ?", token)).One(ctx, a.db)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return roomdb.ErrNotFound
			}
			return err
		}

		if time.Since(session.CreatedAt) > sessionTimeout {
			_, err = session.Delete(ctx, tx)
			if err != nil {
				return err
			}

			return errors.New("sign-in with ssb: session expired")
		}

		memberID = session.MemberID
		return nil
	})
	if err != nil {
		return -1, err
	}

	return memberID, nil
}

// RemoveToken removes a single token from the database
func (a AuthWithSSB) RemoveToken(ctx context.Context, token string) error {
	_, err := models.SIWSSBSessions(qm.Where("token = ?", token)).DeleteAll(ctx, a.db)
	return err
}

// WipeTokensForMember deletes all tokens currently held for that member
func (a AuthWithSSB) WipeTokensForMember(ctx context.Context, memberID int64) error {
	return transact(a.db, func(tx *sql.Tx) error {
		_, err := models.SIWSSBSessions(qm.Where("member_id = ?", memberID)).DeleteAll(ctx, tx)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return roomdb.ErrNotFound
			}
			return err
		}
		return nil
	})
}

// delete sessions that are older then the timeout.
// TODO: maybe change the qm slightly to use the sessionTimeout constant
func deleteExpiredAuthWithSSBSessions(tx boil.ContextExecutor) error {
	_, err := models.SIWSSBSessions(qm.Where("created_at < date('now', '-1 day')")).DeleteAll(context.Background(), tx)
	if err != nil {
		return fmt.Errorf("roomdb: failed to delete expired authWithSSB sessions: %w", err)
	}
	return nil
}
