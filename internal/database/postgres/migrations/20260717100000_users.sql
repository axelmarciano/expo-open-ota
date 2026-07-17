-- +goose Up
-- +goose StatementBegin

CREATE TABLE users (
    id UUID PRIMARY KEY,
    -- Emails are normalized to lowercase in Go before every read and write
    -- (see store.NormalizeEmail), so the plain UNIQUE constraint is enough to
    -- enforce case-insensitive uniqueness and doubles as the login lookup index.
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash TEXT NOT NULL,
    is_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    -- NULL until the first successful sign-in (e.g. the migration-seeded admin
    -- before anyone has logged in).
    last_connected_at TIMESTAMP WITH TIME ZONE
);

-- The "there must always be at least one admin" invariant makes admin counting
-- a hot assertion on every user delete / demote.
CREATE INDEX idx_users_admins ON users (id) WHERE is_admin;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS users;
-- +goose StatementEnd
