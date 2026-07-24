-- +goose Up
-- +goose StatementBegin

-- Supports the branch/channel dashboard summaries: once the newest runtime for
-- a branch is known, its active rollout or latest checked update is read in
-- descending publication order.
CREATE INDEX idx_updates_branch_runtime_created
    ON updates(branch_id, runtime_version_id, created_at DESC, id DESC)
    INCLUDE (commit_hash, rollout_percentage)
    WHERE checked_at IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP INDEX IF EXISTS idx_updates_branch_runtime_created;
-- +goose StatementEnd
