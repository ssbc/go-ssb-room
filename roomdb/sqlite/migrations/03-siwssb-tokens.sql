-- +migrate Up
-- SIWSSB stands for sign-in with ssb
CREATE TABLE SIWSSB_sessions (
  id            INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL,
  token         TEXT UNIQUE NOT NULL,
  member_id     INTEGER NOT NULL,
  created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,

  FOREIGN KEY ( member_id ) REFERENCES members( "id" )
);
CREATE UNIQUE INDEX SIWSSB_by_token ON SIWSSB_sessions(token);
CREATE INDEX SIWSSB_by_member ON SIWSSB_sessions(member_id);

-- +migrate Down
DROP TABLE SIWSSB_sessions;
DROP INDEX SIWSSB_by_token;
DROP INDEX SIWSSB_by_member;