-- +goose Up
-- +goose StatementBegin

-- Append-only audit trail (Enterprise Edition, ee/audit). One row per
-- state-changing action: who did it, what was done, on which resource, with
-- what outcome. Rows are only ever inserted (and purged past the retention
-- window); nothing updates them.
--
-- Deliberately no foreign key to users, apps or api keys: an entry must
-- survive the deletion of everything it references, so actor and target are
-- denormalized snapshots (id + display name) taken at write time. app_id is
-- TEXT for the same reason: it is a fact about the past, not a live reference.
CREATE TABLE audit_log_events (
    id BIGSERIAL PRIMARY KEY,
    occurred_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_type TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    actor_display TEXT NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id TEXT NOT NULL,
    target_display TEXT NOT NULL DEFAULT '',
    app_id TEXT,
    outcome TEXT NOT NULL DEFAULT 'success',
    ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb
);

-- The viewer lists by id DESC (insert order, covered by the primary key) with
-- optional equality filters; each filter index ends with id DESC so a filtered
-- page is one index scan. occurred_at is indexed alone for date-range filters
-- and the retention purge.
CREATE INDEX idx_audit_log_events_actor ON audit_log_events (actor_id, id DESC);
CREATE INDEX idx_audit_log_events_action ON audit_log_events (action, id DESC);
CREATE INDEX idx_audit_log_events_app ON audit_log_events (app_id, id DESC);
CREATE INDEX idx_audit_log_events_occurred ON audit_log_events (occurred_at);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_log_events;
-- +goose StatementEnd
