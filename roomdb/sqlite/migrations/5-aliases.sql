-- +migrate Up
CREATE TABLE aliases (
  id            integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  name          text UNIQUE NOT NULL,
  user_id       integer NOT NULL,
  signature blob not null,

  FOREIGN KEY ( user_id ) REFERENCES allow_list( "id" )
);

CREATE UNIQUE INDEX aliases_ids ON aliases(id);
CREATE UNIQUE INDEX aliases_names ON aliases(name);

-- +migrate Down
DROP TABLE aliases;

DROP INDEX aliases_ids;
DROP INDEX aliases_names;
