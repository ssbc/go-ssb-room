-- +migrate Up
CREATE TABLE allow_list (
  id   integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  pub_key text NOT NULL UNIQUE
);

-- +migrate Down
DROP TABLE allow_list;