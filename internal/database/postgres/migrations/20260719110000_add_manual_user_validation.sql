-- +goose Up
-- +goose StatementBegin

-- Provides a mechanism for admins to manually validate a user account, which is useful for SSO providers
ALTER TABLE users ADD COLUMN enabled BOOLEAN NOT NULL DEFAULT TRUE;
ALTER TABLE sso_config ADD COLUMN manual_user_validation BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE users DROP COLUMN IF EXISTS enabled;
ALTER TABLE sso_config DROP COLUMN IF EXISTS manual_user_validation;
-- +goose StatementEnd