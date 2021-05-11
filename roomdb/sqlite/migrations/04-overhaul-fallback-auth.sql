-- +migrate Up

-- drop login column from fallback pw
-- ==================================

-- this is sqlite style ALTER TABLE abc DROP COLUMN
-- See 5) in https://www.sqlite.org/lang_altertable.html
-- and https://www.sqlitetutorial.net/sqlite-alter-table/

-- drop obsolete index
DROP INDEX fallback_passwords_by_login;

-- create new schema table (without 'login' column)
CREATE TABLE updated_passwords_table (
  id            INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  password_hash BLOB    NOT NULL,

  member_id     INTEGER UNIQUE NOT NULL,

  FOREIGN KEY ( member_id ) REFERENCES members( "id" )  ON DELETE CASCADE
);

-- copy existing values from original table into new
INSERT INTO updated_passwords_table(id, password_hash, member_id)
SELECT id, password_hash, member_id
FROM fallback_passwords;

-- rename the new to the original table name
DROP TABLE fallback_passwords;
ALTER TABLE updated_passwords_table RENAME TO fallback_passwords;

-- create new lookup index by member id
CREATE INDEX fallback_passwords_by_member ON fallback_passwords(member_id);

-- add new table for password reset tokens
--========================================
CREATE TABLE fallback_reset_tokens (
  id               INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  hashed_token     TEXT UNIQUE NOT NULL,
  created_by       INTEGER NOT NULL,
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  for_member        INTEGER NOT NULL,

  active boolean   NOT NULL DEFAULT TRUE,

  FOREIGN KEY ( created_by ) REFERENCES members( "id" )  ON DELETE CASCADE
  FOREIGN KEY ( for_member ) REFERENCES members( "id" )  ON DELETE CASCADE
);
CREATE INDEX fallback_reset_tokens_by_token ON fallback_reset_tokens(hashed_token);

-- +migrate Down
DROP INDEX fallback_passwords_by_member;
DROP TABLE fallback_passwords;

DROP TABLE fallback_reset_tokens;