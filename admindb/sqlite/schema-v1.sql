DROP TABLE IF EXISTS auth_fallback;
CREATE TABLE auth_fallback (
  id   int PRIMARY KEY NOT NULL,
  name text      NOT NULL UNIQUE,
  password_hash blob not null
  -- pub_key text    NOT NULL UNIQUE ????
);

DROP TABLE IF EXISTS aliases;
CREATE TABLE aliases (
  id   int PRIMARY KEY NOT NULL,
  name text NOT NULL UNIQUE,
  pub_key text NOT NULL UNIQUE
);