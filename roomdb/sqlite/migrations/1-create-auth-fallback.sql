-- +migrate Up
CREATE TABLE auth_fallback (
  id   integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  name text      NOT NULL UNIQUE,
  password_hash blob not null  
);

-- +migrate Down
DROP TABLE auth_fallback;