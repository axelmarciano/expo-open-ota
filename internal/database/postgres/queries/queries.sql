-- name: GetAppByID :one
SELECT * FROM apps
WHERE id = $1 LIMIT 1;

-- name: GetApps :many
SELECT id, name 
FROM apps
ORDER BY created_at ASC;

-- name: InsertApp :one
INSERT INTO apps (id, name, keys_mode, sealed_public_key, sealed_private_key, path_public_key, path_private_key, aws_secret_id_public, aws_secret_id_private)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
RETURNING id;

-- name: DeleteAppByID :execresult
DELETE FROM apps
WHERE id = $1;

-- name: UpdateAppNameByID :execresult
UPDATE apps 
SET name = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: InsertChannel :one
INSERT INTO channels (app_id, branch_id, name)
VALUES ($1, $2, $3)
RETURNING id;

-- name: DeleteChannelByName :execresult
DELETE FROM channels
WHERE name = $1 AND app_id = $2;

-- name: GetChannelsByAppID :many
SELECT channels.*, branches.name as branch_name 
FROM channels
LEFT JOIN branches ON channels.branch_id = branches.id AND branches.app_id = channels.app_id
WHERE channels.app_id = $1
ORDER BY channels.created_at ASC;

-- name: GetChannelNamesByBranchName :many
SELECT c.name
FROM channels c
INNER JOIN branches b ON c.branch_id = b.id AND b.app_id = c.app_id
WHERE b.name = $1 AND b.app_id = $2
ORDER BY c.created_at ASC;

-- name: GetChannelBranchMapping :one
SELECT c.id, b.name AS branch_name
FROM channels c
JOIN branches b ON c.branch_id = b.id AND b.app_id = c.app_id
WHERE c.app_id = $1 AND c.name = $2;

-- name: InsertBranch :one
INSERT INTO branches (app_id, name)
VALUES ($1, $2)
RETURNING id;

-- name: GetBranchByName :one
SELECT id FROM branches
WHERE name = $1 AND app_id = $2
LIMIT 1;

-- name: DeleteBranchByName :execresult
DELETE FROM branches
WHERE name = $1 AND app_id = $2;

-- name: GetBranchesByAppID :many
SELECT DISTINCT ON (branches.id) 
    branches.*, 
    channels.name AS channel_name 
FROM branches
LEFT JOIN channels ON branches.id = channels.branch_id AND channels.app_id = branches.app_id
WHERE branches.app_id = $1;

-- name: UpdateChannelBranchMapping :execresult
-- The EXISTS clause scopes the *target* branch to the caller's app. fk_channels_branch
-- only references branches(id), so without it any tenant's branch id satisfies the FK.
UPDATE channels
SET branch_id = $1
WHERE channels.app_id = $2
  AND channels.id = $3
  AND EXISTS (
      SELECT 1 FROM branches
      WHERE branches.id = $1 AND branches.app_id = $2
  );

-- name: GetRuntimeVersionsWithUpdateCount :many
SELECT 
    rv.id, 
    rv.version, 
    rv.created_at, 
    rv.updated_at,
    (
        SELECT COUNT(u.id)
        FROM updates u
        JOIN branches b ON u.branch_id = b.id
        WHERE u.runtime_version_id = rv.id 
          AND b.name = $2 AND u.checked_at IS NOT NULL
    ) AS update_count
FROM runtime_versions rv
WHERE rv.app_id = $1
  -- Only allow rows where at least one matching update exists
  AND EXISTS (
      SELECT 1 
      FROM updates u
      JOIN branches b ON u.branch_id = b.id
      WHERE u.runtime_version_id = rv.id 
        AND b.name = $2
        AND u.checked_at IS NOT NULL
  )
ORDER BY rv.created_at DESC;

-- name: InsertRuntimeVersion :one
INSERT INTO runtime_versions (app_id, version)
VALUES ($1, $2)
RETURNING id;

-- name: GetUpdatesByByBranchNameAndRuntimeVersion :many
SELECT u.id, u.update_uuid, u.update_type, u.created_at, u.commit_hash, u.platform, u.message, u.checked_at
FROM updates u
JOIN runtime_versions rv ON u.runtime_version_id = rv.id
JOIN branches b ON u.branch_id = b.id
JOIN apps a ON b.app_id = a.id
WHERE a.id = $1 
  AND rv.version = $2 
  AND b.name = $3
  AND u.checked_at IS NOT NULL
ORDER BY u.created_at DESC;

-- name: GetUpdateType :one
SELECT u.update_type 
FROM updates u
JOIN branches b ON u.branch_id = b.id
WHERE b.app_id = $1
  AND b.name = $2
  AND u.id = $3;

-- name: GetUpdateCheckedAt :one
SELECT u.checked_at
FROM updates u
JOIN branches b ON u.branch_id = b.id
WHERE b.app_id = $1
  AND b.name = $2
  AND u.id = $3;

-- name: GetUpdateMetadata :one
SELECT updates.id, update_uuid, platform, commit_hash, message
FROM updates
JOIN branches ON updates.branch_id = branches.id
WHERE branches.app_id = $2
  AND branches.name = $3
  AND updates.id = $1;

-- name: StoreUpdateUUID :execresult
UPDATE updates
SET update_uuid = $2
WHERE updates.id = $1 AND branch_id = (
    SELECT branches.id 
    FROM branches 
    WHERE app_id = $3 
      AND name = $4
);

-- name: GetLatestUpdate :one
SELECT 
    u.id,
    u.update_uuid,
    u.branch_id,
    u.runtime_version_id,
    u.update_type,
    u.commit_hash,
    u.message,
    u.platform,
    u.created_at
FROM updates u
JOIN branches b ON u.branch_id = b.id
JOIN runtime_versions rv ON u.runtime_version_id = rv.id
WHERE b.app_id = $1
  AND b.name = $2
  AND rv.version = $3
  AND u.platform = $4
  AND u.checked_at IS NOT NULL
ORDER BY u.id DESC
LIMIT 1;

-- name: GetUpdateByBranchNameAndRuntime :one
-- app_id is load-bearing, not redundant: pk_updates is (branch_id, id), so an
-- update id is only unique per branch, and branch names are only unique per app.
-- Without the app filter the same (id, branch, runtime) triple matches another
-- tenant's row.
SELECT u.id, u.update_uuid, b.app_id, b.name AS branch_name, r.version AS runtime_version, u.update_type, u.commit_hash, u.message, u.platform, u.created_at
FROM updates u
INNER JOIN branches b ON u.branch_id = b.id
INNER JOIN runtime_versions r ON u.runtime_version_id = r.id
WHERE b.app_id = $1
  AND u.id = $2
  AND b.name = $3
  AND r.version = $4
LIMIT 1;

-- name: GetUpdatesMetadataByBranchName :many
SELECT u.id, rv.version AS runtime_version
FROM updates u
INNER JOIN branches b ON u.branch_id = b.id
INNER JOIN runtime_versions rv ON u.runtime_version_id = rv.id
WHERE b.name = $1 AND b.app_id = $2;

-- name: MarkUpdateAsChecked :exec
WITH updated_rows AS (
    UPDATE updates
    SET checked_at = CURRENT_TIMESTAMP
    WHERE updates.id = $1
      AND updates.branch_id = (
          SELECT branches.id 
          FROM branches 
          WHERE branches.app_id = $2
            AND branches.name = $3
      )
    RETURNING runtime_version_id
)
UPDATE runtime_versions
SET updated_at = CURRENT_TIMESTAMP
WHERE id = (SELECT runtime_version_id FROM updated_rows);

-- name: InsertUpdate :one
WITH resolved_names AS (
    SELECT 
        b.id AS resolved_branch_id,
        rv.id AS resolved_runtime_version_id,
        b.app_id,
        b.name AS branch_name,
        rv.version AS runtime_version
    FROM branches b
    INNER JOIN runtime_versions rv ON rv.app_id = b.app_id
    WHERE b.name = $2
      AND rv.version = $4
      AND b.app_id = $3
)
INSERT INTO updates (
    id, 
    branch_id, 
    runtime_version_id, 
    update_type, 
    platform, 
    commit_hash, 
    message
) VALUES (
    $1,
    (SELECT resolved_branch_id FROM resolved_names),
    (SELECT resolved_runtime_version_id FROM resolved_names),
    $5,
    $6,
    $7,
    $8
)
RETURNING 
    id, 
    platform, 
    commit_hash, 
    message, 
    created_at,
    (SELECT app_id FROM resolved_names) AS app_id,
    (SELECT branch_name FROM resolved_names) AS branch_name,
    (SELECT runtime_version FROM resolved_names) AS runtime_version;

-- name: InsertApiKey :exec
INSERT INTO api_keys (app_id, name, hint, hashed_key)
VALUES ($1, $2, $3, $4);

-- name: GetApiKeysMetadataByAppID :many
SELECT id, name, hint, created_at, last_used_at
FROM api_keys
WHERE app_id = $1 AND revoked_at IS NULL
ORDER BY created_at ASC;

-- name: RevokeApiKeyByID :execresult
UPDATE api_keys
SET revoked_at = CURRENT_TIMESTAMP
WHERE id = $1 AND app_id = $2;

-- name: ValidateAndTouchAuth :one
-- Returns the matched key id so the caller can enforce per-key restrictions
-- (enterprise) on top of the authentication itself.
UPDATE api_keys
SET last_used_at = CURRENT_TIMESTAMP
WHERE app_id = $1
  AND hashed_key = $2
  AND revoked_at IS NULL
RETURNING id;

-- name: InsertUser :one
INSERT INTO users (id, email, password_hash, is_admin)
VALUES ($1, $2, $3, $4)
RETURNING id, email, is_admin, created_at;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUsers :many
SELECT id, email, is_admin, created_at, last_connected_at FROM users
ORDER BY created_at ASC;

-- name: TouchUserLastConnectedAt :exec
UPDATE users
SET last_connected_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUserByID :execresult
-- Locks the admin rows first so concurrent deletes/demotions serialize:
-- deleting the last remaining admin matches no row instead of leaving the
-- dashboard without any admin.
WITH admins AS (
    SELECT id FROM users WHERE is_admin ORDER BY id FOR UPDATE
)
DELETE FROM users
WHERE users.id = $1
  AND (users.id NOT IN (SELECT id FROM admins) OR (SELECT COUNT(*) FROM admins) > 1);

-- name: UpdateUserPasswordByID :execresult
UPDATE users
SET password_hash = $2, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: UpdateUserIsAdminByID :execresult
-- Same admin-row lock as DeleteUserByID: demoting the last remaining admin
-- matches no row. Promotions ($2 true) always pass the guard but still take
-- the lock, so they serialize with concurrent demotions.
WITH admins AS (
    SELECT id FROM users WHERE is_admin ORDER BY id FOR UPDATE
)
UPDATE users
SET is_admin = $2, updated_at = CURRENT_TIMESTAMP
WHERE users.id = $1
  AND ($2::boolean
       OR users.id NOT IN (SELECT id FROM admins)
       OR (SELECT COUNT(*) FROM admins) > 1);

-- name: MigrateLegacyApp :exec
INSERT INTO apps (
    id, 
    name, 
    keys_mode, 
    sealed_public_key, 
    sealed_private_key, 
    path_public_key, 
    path_private_key, 
    aws_secret_id_public, 
    aws_secret_id_private
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
ON CONFLICT (id) DO UPDATE SET
    name = EXCLUDED.name,
    keys_mode = EXCLUDED.keys_mode,
    sealed_public_key = EXCLUDED.sealed_public_key,
    sealed_private_key = EXCLUDED.sealed_private_key,
    path_public_key = EXCLUDED.path_public_key,
    path_private_key = EXCLUDED.path_private_key,
    aws_secret_id_public = EXCLUDED.aws_secret_id_public,
    aws_secret_id_private = EXCLUDED.aws_secret_id_private;

-- name: MigrateLegacyChannel :exec
INSERT INTO channels (
    app_id, 
    branch_id, 
    name
) VALUES (
    $1, 
    $2, 
    $3
)
ON CONFLICT (app_id, name) DO UPDATE SET
    branch_id = EXCLUDED.branch_id;

-- name: MigrateLegacyBranch :one
INSERT INTO branches (
    app_id, 
    name
) VALUES (
    $1, 
    $2
)
ON CONFLICT (app_id, name) DO UPDATE SET
    name = EXCLUDED.name
RETURNING id;

-- name: MigrateLegacyRuntimeVersion :exec
INSERT INTO runtime_versions (
    app_id, 
    version, 
    created_at, 
    updated_at
) VALUES (
    $1, 
    $2, 
    $3, 
    $4
)
ON CONFLICT (app_id, version) DO UPDATE SET
    updated_at = EXCLUDED.updated_at;

-- name: MigrateLegacyUpdate :exec
INSERT INTO updates (
    id, 
    branch_id, 
    runtime_version_id, 
    update_type, 
    platform, 
    commit_hash, 
    message,
    checked_at,
    update_uuid,
    created_at
) VALUES (
    $1,
    (SELECT id FROM branches b WHERE b.app_id = $2 AND b.name = $3),
    (SELECT id FROM runtime_versions rv WHERE rv.app_id = $2 AND rv.version = $4),
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11
)
ON CONFLICT (branch_id, id) DO UPDATE SET
    runtime_version_id = EXCLUDED.runtime_version_id,
    update_type = EXCLUDED.update_type,
    platform = EXCLUDED.platform,
    commit_hash = EXCLUDED.commit_hash,
    message = EXCLUDED.message,
    checked_at = EXCLUDED.checked_at,
    update_uuid = EXCLUDED.update_uuid,
    created_at = EXCLUDED.created_at;
-- name: GetEnterpriseLicense :one
SELECT * FROM enterprise_license
WHERE singleton;

-- name: UpsertEnterpriseLicense :one
INSERT INTO enterprise_license (singleton, license_key)
VALUES (TRUE, $1)
ON CONFLICT (singleton) DO UPDATE
SET license_key = EXCLUDED.license_key, updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteEnterpriseLicense :exec
DELETE FROM enterprise_license;

-- name: GetSSOConfig :one
SELECT * FROM sso_config
WHERE singleton;

-- name: UpsertSSOConfig :one
INSERT INTO sso_config (singleton, issuer, client_id, sealed_client_secret, provider_name, scopes, enabled, allowed_email_domains, allowed_groups, groups_claim, trust_unverified_email)
VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
ON CONFLICT (singleton) DO UPDATE
SET issuer = EXCLUDED.issuer,
    client_id = EXCLUDED.client_id,
    sealed_client_secret = EXCLUDED.sealed_client_secret,
    provider_name = EXCLUDED.provider_name,
    scopes = EXCLUDED.scopes,
    enabled = EXCLUDED.enabled,
    allowed_email_domains = EXCLUDED.allowed_email_domains,
    allowed_groups = EXCLUDED.allowed_groups,
    groups_claim = EXCLUDED.groups_claim,
    trust_unverified_email = EXCLUDED.trust_unverified_email,
    updated_at = CURRENT_TIMESTAMP
RETURNING *;

-- name: DeleteSSOConfig :exec
DELETE FROM sso_config;

-- name: GetUserBySSOSubject :one
SELECT u.* FROM users u
JOIN sso_identities si ON si.user_id = u.id
WHERE si.issuer = $1 AND si.subject = $2;

-- name: InsertSSOIdentity :exec
INSERT INTO sso_identities (issuer, subject, user_id, email)
VALUES ($1, $2, $3, $4);

-- name: TouchSSOIdentityLastLogin :exec
UPDATE sso_identities
SET last_login_at = CURRENT_TIMESTAMP
WHERE issuer = $1 AND subject = $2;

-- The queries below back the Enterprise Edition per-key access restrictions
-- (ee/apikeyrestrictions). sqlc generates a single package for the whole
-- schema, so the EE feature's SQL lives here like the enterprise license
-- queries above.

-- name: GetApiKeyRestrictions :one
-- Enforcement read for one authenticated key on the CLI request hot path.
SELECT allowed_ips, can_access_protected_branches
FROM api_keys
WHERE id = $1;

-- name: GetApiKeyRestrictionsByAppID :many
SELECT id, allowed_ips, can_access_protected_branches
FROM api_keys
WHERE app_id = $1 AND revoked_at IS NULL;

-- name: UpdateApiKeyRestrictions :execrows
UPDATE api_keys
SET allowed_ips = $1, can_access_protected_branches = $2
WHERE id = $3 AND app_id = $4 AND revoked_at IS NULL;

-- name: SetBranchProtected :execrows
UPDATE branches
SET protected = $1
WHERE app_id = $2 AND name = $3;

-- name: IsBranchProtected :one
SELECT protected FROM branches
WHERE app_id = $1 AND name = $2;
