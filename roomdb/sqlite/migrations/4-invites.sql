-- +migrate Up
CREATE TABLE invites (
  id               integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  token            text UNIQUE NOT NULL,
  created_by       integer NOT NULL,
  alias_suggestion text NOT NULL DEFAULT "", -- optional
  active boolean NOT NULL DEFAULT true,

  -- TODO: replace auth_fallback with a user table once we do "sign in with ssb"
  FOREIGN KEY ( created_by ) REFERENCES auth_fallback( "id" )
);

CREATE INDEX invite_active_ids ON invites(id) WHERE active=true;
CREATE UNIQUE INDEX invite_active_tokens ON invites(token) WHERE active=true;
CREATE INDEX invite_inactive ON invites(active);

-- +migrate Down
DROP TABLE invites;

DROP INDEX invite_active_ids;
DROP INDEX invite_active_tokens;
DROP INDEX invite_inactive;
