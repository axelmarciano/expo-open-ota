-- +goose Up

-- One eoas publish produces one update row per platform (iOS then Android),
-- each with its own id and uuid; nothing ties them together. The CLI now mints
-- a single UUID per publish run and sends it with each per-platform call
-- (group republishes get a server-minted one), stored here so consumers
-- (dashboard grouping, per-publish health) can treat the set as one publish.
-- NULL for rows created by older CLIs, rollback markers (branch-level
-- operations, never grouped) and stateless mode, which degrade to the
-- ungrouped per-platform display. No index: readers fetch the branch listing
-- and group in memory; add one when a query filters on it.
ALTER TABLE updates ADD COLUMN publish_group UUID;

-- +goose Down
ALTER TABLE updates DROP COLUMN publish_group;
