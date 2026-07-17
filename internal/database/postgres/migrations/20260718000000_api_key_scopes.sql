-- +goose Up
-- +goose StatementBegin

-- Channel scoping for API keys. A key with rows here is only valid for the
-- listed channels; a key with no rows keeps the current behavior (all
-- channels allowed). App consistency (key and channel belonging to the same
-- app) is enforced by the insert query, not by a composite FK.
CREATE TABLE api_key_channels (
    api_key_id BIGINT NOT NULL,
    channel_id BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT pk_api_key_channels PRIMARY KEY (api_key_id, channel_id),
    CONSTRAINT fk_api_key_channels_key FOREIGN KEY (api_key_id) REFERENCES api_keys(id) ON DELETE CASCADE,
    CONSTRAINT fk_api_key_channels_channel FOREIGN KEY (channel_id) REFERENCES channels(id) ON DELETE CASCADE
);

CREATE INDEX idx_api_key_channels_channel_id ON api_key_channels(channel_id);

-- IP whitelist in CIDR notation (e.g. 203.0.113.0/24, 2001:db8::/32).
-- NULL or empty array = no IP restriction.
ALTER TABLE api_keys ADD COLUMN allowed_ips CIDR[];

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE api_keys DROP COLUMN IF EXISTS allowed_ips;
DROP TABLE IF EXISTS api_key_channels;
-- +goose StatementEnd
