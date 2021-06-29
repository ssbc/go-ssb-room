-- +migrate Up
CREATE TABLE pins (
  id integer NOT NULL PRIMARY KEY,
  name text NOT NULL UNIQUE
);


CREATE TABLE notices (
  id   integer PRIMARY KEY AUTOINCREMENT NOT NULL,
  title text NOT NULL,
  content text NOT NULL,
  language text NOT NULL
);

-- n:m relation table
CREATE TABLE pin_notices (
  notice_id integer NOT NULL,
  pin_id integer NOT NULL,
  
  PRIMARY KEY (notice_id, pin_id),

  FOREIGN KEY ( notice_id ) REFERENCES notices( "id" ),
  FOREIGN KEY ( pin_id ) REFERENCES pins( "id" )
);

-- +migrate Down
DROP TABLE notices;
DROP TABLE pins;
DROP TABLE pin_notices;