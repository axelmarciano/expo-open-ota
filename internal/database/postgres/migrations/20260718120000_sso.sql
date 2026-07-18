-- +goose Up
-- +goose StatementBegin

-- Single sign-on configuration (Enterprise Edition, ee/sso). One OIDC
-- provider per server: the CHECK-ed singleton column caps the table at a
-- single row, edited from the dashboard's License page. The client secret is
-- sealed with AES-GCM under the deployment's DB keys master key (the AAD is
-- derived in ee/sso, never stored); every other field is plain configuration.
CREATE TABLE sso_config (
    singleton BOOLEAN PRIMARY KEY DEFAULT TRUE CHECK (singleton),
    issuer TEXT NOT NULL,
    client_id TEXT NOT NULL,
    sealed_client_secret TEXT NOT NULL,
    provider_name TEXT NOT NULL DEFAULT 'SSO',
    scopes TEXT NOT NULL DEFAULT 'openid profile email',
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    -- Optional sign-in restrictions, empty by default (any account the IdP
    -- authenticates may sign in). Domains are matched against the email's
    -- domain part; groups against the values of the groups_claim claim of
    -- the id_token.
    allowed_email_domains TEXT[] NOT NULL DEFAULT '{}',
    allowed_groups TEXT[] NOT NULL DEFAULT '{}',
    groups_claim TEXT NOT NULL DEFAULT 'groups',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Maps an OIDC identity (issuer, subject) to a dashboard user. The subject is
-- the stable identifier: the email at the IdP can change without losing the
-- account. Deleting a user cascades here; the next SSO sign-in of that person
-- simply provisions a fresh member account.
CREATE TABLE sso_identities (
    issuer TEXT NOT NULL,
    subject TEXT NOT NULL,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_login_at TIMESTAMP WITH TIME ZONE,
    PRIMARY KEY (issuer, subject)
);

CREATE INDEX idx_sso_identities_user ON sso_identities (user_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS sso_identities;
DROP TABLE IF EXISTS sso_config;
-- +goose StatementEnd
