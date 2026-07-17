-- Deleting an app must remove all of its owned data. api_keys, branches and
-- updates already cascade from apps, but channels and runtime_versions were
-- created with ON DELETE RESTRICT, so deleting an app with any channel or
-- runtime version failed with:
--   update or delete on table "apps" violates RESTRICT setting of foreign key
--   constraint "fk_channels_app" on table "channels" (SQLSTATE 23001)
--
-- Fix: make the app-owned FKs cascade like the others, and relax the two
-- sibling FKs (channel->branch, update->runtime_version) from RESTRICT to
-- NO ACTION. NO ACTION is a *deferred* RESTRICT: it still blocks a direct
-- branch/runtime-version delete that would orphan a channel/update, but it
-- lets the app-level cascade remove those children first within the same
-- statement instead of aborting on cascade-processing order.

-- +goose Up
ALTER TABLE channels DROP CONSTRAINT fk_channels_app;
ALTER TABLE channels ADD CONSTRAINT fk_channels_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE channels DROP CONSTRAINT fk_channels_branch;
ALTER TABLE channels ADD CONSTRAINT fk_channels_branch FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE NO ACTION;

ALTER TABLE runtime_versions DROP CONSTRAINT fk_runtime_versions_app;
ALTER TABLE runtime_versions ADD CONSTRAINT fk_runtime_versions_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE;

ALTER TABLE updates DROP CONSTRAINT fk_updates_runtime_version;
ALTER TABLE updates ADD CONSTRAINT fk_updates_runtime_version FOREIGN KEY (runtime_version_id) REFERENCES runtime_versions(id) ON DELETE NO ACTION;

-- +goose Down
ALTER TABLE channels DROP CONSTRAINT fk_channels_app;
ALTER TABLE channels ADD CONSTRAINT fk_channels_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE RESTRICT;

ALTER TABLE channels DROP CONSTRAINT fk_channels_branch;
ALTER TABLE channels ADD CONSTRAINT fk_channels_branch FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE RESTRICT;

ALTER TABLE runtime_versions DROP CONSTRAINT fk_runtime_versions_app;
ALTER TABLE runtime_versions ADD CONSTRAINT fk_runtime_versions_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE RESTRICT;

ALTER TABLE updates DROP CONSTRAINT fk_updates_runtime_version;
ALTER TABLE updates ADD CONSTRAINT fk_updates_runtime_version FOREIGN KEY (runtime_version_id) REFERENCES runtime_versions(id) ON DELETE RESTRICT;
