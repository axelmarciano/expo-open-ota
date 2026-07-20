-- +goose Up
-- +goose StatementBegin

-- Stores the Enterprise Edition license key (control-plane deployments only).
-- Exactly one license per server: the CHECK-ed singleton column caps the table
-- at a single row, and replacing the key is an upsert on that row. The key is
-- stored verbatim ("key/{dataset}.{signature}"); it is a signed public payload
-- verified offline by ee/licensing, not a secret.
CREATE TABLE enterprise_license (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    license_key TEXT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS enterprise_license;
-- +goose StatementEnd
