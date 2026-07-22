-- +goose Up
-- +goose StatementBegin

-- Cursor of the audit archive exporter (ee/audit): the highest audit event id
-- already written to the dedicated archive bucket. A singleton row, advanced
-- with an optimistic compare-and-swap so multi-replica deployments never
-- double-advance: replicas racing the same batch write the same file (same
-- key, same content, an idempotent overwrite) and only one wins the advance.
CREATE TABLE audit_export_state (
    id BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (id),
    last_exported_id BIGINT NOT NULL DEFAULT 0
);
INSERT INTO audit_export_state (id) VALUES (TRUE);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS audit_export_state;
-- +goose StatementEnd
