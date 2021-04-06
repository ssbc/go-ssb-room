-- +migrate Up
-- the configuration settings for this room, currently only privacy modes
CREATE TABLE config (
    id integer NOT NULL PRIMARY KEY,
    privacyMode integer NOT NULL,    -- open, community, restricted

    CHECK (id == 0) -- should only ever store one row
);

-- the config table will only ever contain one row: the rooms current settings
-- we update that row whenever the config changes.
-- to have something to update, we insert the first and only row at id 0
INSERT INTO config (id, privacyMode) VALUES (
    0,  -- the constant id we will query
    1   -- community is the default mode unless overridden
);

-- +migrate Down
DROP TABLE config;
