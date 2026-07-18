-- +goose Up
-- +goose StatementBegin

-- Progressive rollouts (MIT core, control-plane / Postgres mode only).

-- Channel rollout: a channel serves a rollout branch to `percentage`% of devices
-- and its mapped (default) branch to the rest. One active rollout per channel is
-- DB-enforced via the UNIQUE(channel_id). The row id doubles as the bucketing salt
-- (generated in Go). Deleting the channel ends its rollout (CASCADE). The branch FK is
-- NO ACTION DEFERRABLE INITIALLY DEFERRED: a direct DELETE of a branch serving a
-- rollout still fails at commit (the reference survives), but deleting an app passes,
-- because its apps->channels->channel_rollouts cascade removes the referencing row
-- before the deferred check runs. A non-deferred constraint would abort app deletion:
-- the apps->branches cascade fires the branch check before the channel cascade has
-- reached channel_rollouts (same failure mode 20260614000000_app_delete_cascade.sql
-- fixed for fk_channels_branch, one level deeper).
CREATE TABLE channel_rollouts (
    id UUID PRIMARY KEY,
    channel_id BIGINT NOT NULL UNIQUE,
    rollout_branch_id BIGINT NOT NULL,
    percentage INT NOT NULL CHECK (percentage BETWEEN 1 AND 99),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_channel_rollouts_channel FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE,
    CONSTRAINT fk_channel_rollouts_branch FOREIGN KEY (rollout_branch_id) REFERENCES branches(id) DEFERRABLE INITIALLY DEFERRED
);

CREATE INDEX idx_channel_rollouts_branch ON channel_rollouts(rollout_branch_id);

-- Per-update rollout: each per-platform update row carries its own state. An update
-- with rollout_percentage set is served to that fraction of devices; the rest receive
-- control_update_id (the previous checked update of the same branch/rtv/platform, or
-- NULL for the first update of a branch).
ALTER TABLE updates ADD COLUMN rollout_percentage INT CHECK (rollout_percentage IS NULL OR rollout_percentage BETWEEN 1 AND 99);
ALTER TABLE updates ADD COLUMN control_update_id BIGINT;

-- The control is always on the same branch, so the FK is composite against the
-- (branch_id, id) primary key. control_update_id is nullable and MATCH SIMPLE skips
-- the check when it is NULL. NO ACTION (the default) lets the branch-delete cascade
-- pass: the whole set of a branch's updates is removed in one statement, so no
-- dangling reference survives at statement end.
ALTER TABLE updates ADD CONSTRAINT fk_updates_control FOREIGN KEY (branch_id, control_update_id) REFERENCES updates(branch_id, id);

-- At most one ACTIVE per-update rollout per (branch, rtv, platform). "Active" is
-- rollout_percentage IS NOT NULL AND checked_at IS NOT NULL everywhere: the dedupe-406
-- path leaves an unchecked row whose rollout fields are therefore inert. The per-platform
-- index is required because eoas publishes one update row per (rtv, platform).
CREATE UNIQUE INDEX uq_updates_active_rollout ON updates(branch_id, runtime_version_id, platform)
    WHERE rollout_percentage IS NOT NULL AND checked_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS uq_updates_active_rollout;
ALTER TABLE updates DROP CONSTRAINT IF EXISTS fk_updates_control;
ALTER TABLE updates DROP COLUMN IF EXISTS control_update_id;
ALTER TABLE updates DROP COLUMN IF EXISTS rollout_percentage;
DROP TABLE IF EXISTS channel_rollouts;
-- +goose StatementEnd
