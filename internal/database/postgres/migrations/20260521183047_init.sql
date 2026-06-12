-- +goose Up
-- +goose StatementBegin

CREATE TABLE apps (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    keys_mode VARCHAR(100),
    sealed_public_key TEXT,
    sealed_private_key TEXT,
    path_public_key TEXT,
    path_private_key TEXT,
    aws_secret_id_public VARCHAR(255),
    aws_secret_id_private VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);


CREATE TABLE api_keys (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    hint VARCHAR(32) NOT NULL,
    hashed_key CHAR(64) NOT NULL UNIQUE, -- SHA-256 hash
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    last_used_at TIMESTAMP WITH TIME ZONE,
    revoked_at TIMESTAMP WITH TIME ZONE,
    CONSTRAINT fk_api_keys_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE
);

CREATE TABLE branches (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_branches_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE CASCADE,
    CONSTRAINT uq_branch_per_app UNIQUE (app_id, name)
);

CREATE TABLE channels (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL,
    branch_id BIGINT,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_channels_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE RESTRICT,
    CONSTRAINT fk_channels_branch FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE RESTRICT,
    CONSTRAINT uq_channel_per_app UNIQUE (app_id, name)
);

CREATE TABLE runtime_versions (
    id BIGSERIAL PRIMARY KEY,
    app_id UUID NOT NULL,
    version VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_runtime_versions_app FOREIGN KEY (app_id) REFERENCES apps(id) ON DELETE RESTRICT,
    CONSTRAINT uq_version_per_app UNIQUE (app_id, version)
);

CREATE TABLE updates (
    id BIGINT NOT NULL,
    update_uuid UUID,
    branch_id BIGINT NOT NULL,
    runtime_version_id BIGINT NOT NULL,
    update_type INT NOT NULL,
    commit_hash VARCHAR(255) NOT NULL,
    message TEXT,
    platform VARCHAR(100) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    checked_at TIMESTAMP WITH TIME ZONE,

    -- Composite Primary Key (Id is unique per branch)
    CONSTRAINT pk_updates PRIMARY KEY (branch_id, id),
    
    CONSTRAINT fk_updates_branch FOREIGN KEY (branch_id) REFERENCES branches(id) ON DELETE CASCADE,
    CONSTRAINT fk_updates_runtime_version FOREIGN KEY (runtime_version_id) REFERENCES runtime_versions(id) ON DELETE RESTRICT
);

CREATE INDEX idx_api_keys_lookup ON api_keys(app_id, hashed_key) WHERE revoked_at IS NULL;
CREATE INDEX idx_branches_name_app_id ON branches(name, app_id);
CREATE INDEX idx_channels_branch_id ON channels(branch_id);

CREATE INDEX idx_updates_rv_branch_checked ON updates(runtime_version_id, branch_id) WHERE checked_at IS NOT NULL;
CREATE INDEX idx_updates_latest_lookup ON updates (branch_id, platform, id DESC) WHERE checked_at IS NOT NULL;
CREATE INDEX idx_updates_runtime_version_id ON updates(runtime_version_id);

-- Enforces unique UUIDs for normal updates while allowing multiple NULL Expo rollbacks
CREATE UNIQUE INDEX uq_updates_uuid_conditional ON updates(update_uuid) WHERE update_uuid IS NOT NULL;

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS updates;
DROP TABLE IF EXISTS channels;
DROP TABLE IF EXISTS branches;
DROP TABLE IF EXISTS runtime_versions;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS apps;
-- +goose StatementEnd