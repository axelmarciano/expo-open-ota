-- +goose Up
-- +goose StatementBegin

-- Whether the deployment trusts the email a provider asserts even when the
-- id_token does not carry email_verified=true. FALSE by default so an
-- unverified (or absent) email cannot be used for domain authorization or to
-- link/provision an account: without this an attacker able to set an
-- arbitrary unverified email at the IdP could take over an existing account
-- by matching its address. Admins turn it on knowingly for a trusted provider
-- that omits email_verified (notably Microsoft Entra ID) on a single tenant
-- where users cannot self-assert addresses.
ALTER TABLE sso_config ADD COLUMN trust_unverified_email BOOLEAN NOT NULL DEFAULT FALSE;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE sso_config DROP COLUMN IF EXISTS trust_unverified_email;
-- +goose StatementEnd
