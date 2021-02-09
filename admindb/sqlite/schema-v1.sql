-- TODO: unify user table of auth_fallback and auth_signin_with_ssb


DROP TABLE IF EXISTS aliases;
CREATE TABLE aliases (
  id   int PRIMARY KEY NOT NULL,
  name text NOT NULL UNIQUE,
  pub_key text NOT NULL UNIQUE
);