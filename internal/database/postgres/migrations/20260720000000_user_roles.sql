-- +goose Up
-- +goose StatementBegin

-- Named permission bundles (Enterprise Edition, ee/rbac). A role is global:
-- it describes what a member may do on an app; which apps it applies to is
-- decided per user in user_app_grants. The permission strings are validated
-- in Go against the ee/rbac catalog before every write.
CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL UNIQUE,
    permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- One row = one member's access to one app. No row, no access: with an active
-- enterprise license members only see the apps they are granted (admins bypass
-- the whole table). Effective permissions are the union of the role's
-- permissions and extra_permissions; either side may be empty, a row with both
-- empty still grants read access to the app.
CREATE TABLE user_app_grants (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    app_id UUID NOT NULL REFERENCES apps(id) ON DELETE CASCADE,
    -- RESTRICT: deleting a role that is still assigned must fail loudly
    -- instead of silently stripping permissions from members.
    role_id UUID REFERENCES roles(id) ON DELETE RESTRICT,
    extra_permissions TEXT[] NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, app_id)
);

CREATE INDEX idx_user_app_grants_app ON user_app_grants (app_id);
CREATE INDEX idx_user_app_grants_role ON user_app_grants (role_id);

-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
DROP TABLE IF EXISTS user_app_grants;
DROP TABLE IF EXISTS roles;
-- +goose StatementEnd
