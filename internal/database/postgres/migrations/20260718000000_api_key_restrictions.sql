-- +goose Up
-- +goose StatementBegin

-- Per-key access restrictions (Enterprise Edition, ee/apikeyrestrictions).

-- IP allowlist in CIDR notation (e.g. 203.0.113.0/24, 2001:db8::/32).
-- NULL or empty array = no IP restriction.
ALTER TABLE api_keys ADD COLUMN allowed_ips CIDR[];

-- Whether the key may publish, rollback or republish on protected branches.
-- FALSE by default: a fresh key never touches protected branches until an
-- admin explicitly grants it.
ALTER TABLE api_keys ADD COLUMN can_access_protected_branches BOOLEAN NOT NULL DEFAULT FALSE;

-- GitHub-style branch protection, toggled by admins from the dashboard.
ALTER TABLE branches ADD COLUMN protected BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE branches DROP COLUMN IF EXISTS protected;
ALTER TABLE api_keys DROP COLUMN IF EXISTS can_access_protected_branches;
ALTER TABLE api_keys DROP COLUMN IF EXISTS allowed_ips;
-- +goose StatementEnd
