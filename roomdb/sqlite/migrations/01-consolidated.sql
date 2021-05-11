-- +migrate Up
-- the internal users table (people who used an invite)
CREATE TABLE members (
  id            INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  role          INTEGER NOT NULL, -- member, moderator or admin
  pub_key       TEXT    NOT NULL UNIQUE,

  CHECK(role > 0)
);
CREATE INDEX members_pubkeys ON members(pub_key);

-- password login for members (in case they can't use sign-in with ssb, for whatever reason)
CREATE TABLE fallback_passwords (
  id            INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  login         TEXT    NOT NULL UNIQUE,
  password_hash BLOB    NOT NULL,

  member_id     INTEGER NOT NULL,

  FOREIGN KEY ( member_id ) REFERENCES members( "id" )  ON DELETE CASCADE
);
CREATE INDEX fallback_passwords_by_login ON fallback_passwords(login);

-- single use tokens for becoming members
CREATE TABLE invites (
  id               INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  hashed_token     TEXT UNIQUE NOT NULL,
  created_by       INTEGER NOT NULL,
  created_at       DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  active boolean   NOT NULL DEFAULT TRUE,

  FOREIGN KEY ( created_by ) REFERENCES members( "id" )  ON DELETE CASCADE
);
CREATE INDEX invite_active_ids ON invites(id) WHERE active=TRUE;
CREATE UNIQUE INDEX invite_active_tokens ON invites(hashed_token) WHERE active=TRUE;
CREATE INDEX invite_inactive ON invites(active);

-- name -> public key mappings
CREATE TABLE aliases (
  id            INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  name          TEXT UNIQUE NOT NULL,
  member_id     INTEGER NOT NULL,
  signature     BLOB NOT NULL,

  FOREIGN KEY ( member_id ) REFERENCES members( "id" )  ON DELETE CASCADE
);
CREATE UNIQUE INDEX aliases_ids ON aliases(id);
CREATE UNIQUE INDEX aliases_names ON aliases(name);

-- public keys that should never ever be let into the room
CREATE TABLE denied_keys (
  id          INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  pub_key     TEXT NOT NULL UNIQUE,
  comment     TEXT NOT NULL,
  created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX denied_keys_by_pubkey ON invites(active);

-- +migrate Down
DROP TABLE members;

DROP INDEX fallback_passwords_by_login;
DROP TABLE fallback_passwords;

DROP INDEX invite_active_ids;
DROP INDEX invite_active_tokens;
DROP INDEX invite_inactive;
DROP TABLE invites;

DROP INDEX aliases_ids;
DROP INDEX aliases_names;
DROP TABLE aliases;

DROP INDEX denied_keys_by_pubkey;
DROP TABLE denied_keys;