-- +migrate Up
-- the configuration settings for this room, currently privacy mode settings and the default translation for the room
CREATE TABLE config (
    id integer NOT NULL PRIMARY KEY,
    privacyMode integer NOT NULL,    -- open, community, restricted
    defaultLanguage TEXT NOT NULL,   -- a language tag, e.g. en, sv, de
    use_subdomain_for_aliases boolean NOT NULL, -- flag to toggle using subdomains (rather than alias routes) for aliases

    CHECK (id == 0) -- should only ever store one row
);

-- the config table will only ever contain one row: the rooms current settings
-- we update that row whenever the config changes.
-- to have something to update, we insert the first and only row at id 0
INSERT INTO config (id, privacyMode, defaultLanguage, use_subdomain_for_aliases) VALUES (
    0,     -- the constant id we will query
    2,     -- community is the default mode unless overridden
    "en",  -- english is the default language for all installs
    1      -- use subdomain for aliases
);

-- +migrate Down
DROP TABLE config;
