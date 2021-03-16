-- +migrate Up
ALTER TABLE invites ADD COLUMN created_at datetime NOT NULL DEFAULT 0;
-- because of the restrictions in 4. https://www.sqlite.org/lang_altertable.html
-- the actual default should be date('now') but can't be.
-- this needs to be fixed once we merge the migrations into a signle schema prior to the next point release.
-- at which point we dont need to insert the current time into the db from the application anymore.
UPDATE invites set created_at=date('now');

-- +migrate Down
ALTER TABLE invites DROP COLUMN created_at;