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
SELECT channels.*, branches.name as branch_name,
    cr.id AS rollout_id,
    rb.name AS rollout_branch_name,
    cr.percentage AS rollout_percentage,
    cr.created_at AS rollout_created_at,
    cr.updated_at AS rollout_updated_at
FROM channels
LEFT JOIN branches ON channels.branch_id = branches.id AND branches.app_id = channels.app_id
LEFT JOIN channel_rollouts cr ON cr.channel_id = channels.id
LEFT JOIN branches rb ON cr.rollout_branch_id = rb.id
WHERE channels.app_id = $1
ORDER BY channels.created_at ASC;

-- name: GetChannelNamesByBranchName :many
SELECT c.name
FROM channels c
INNER JOIN branches b ON c.branch_id = b.id AND b.app_id = c.app_id
WHERE b.name = $1 AND b.app_id = $2
ORDER BY c.created_at ASC;

-- name: GetChannelBranchMapping :one
-- Hot path (manifest resolution). The LEFT JOINs fold the channel's active rollout
-- (if any) into the single mapping read so branch resolution stays ONE query.
SELECT c.id, b.name AS branch_name,
    cr.id AS rollout_id,
    rb.name AS rollout_branch_name,
    cr.percentage AS rollout_percentage
FROM channels c
JOIN branches b ON c.branch_id = b.id AND b.app_id = c.app_id
LEFT JOIN channel_rollouts cr ON cr.channel_id = c.id
LEFT JOIN branches rb ON cr.rollout_branch_id = rb.id
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
-- The NOT EXISTS clause refuses to remap a channel while it has an active rollout
-- (the mapping is locked until the rollout is promoted or reverted). Promotion repoints
-- the channel through RepointChannelToRolloutBranch instead, so it is not blocked here.
UPDATE channels
SET branch_id = $1
WHERE channels.app_id = $2
  AND channels.id = $3
  AND EXISTS (
      SELECT 1 FROM branches
      WHERE branches.id = $1 AND branches.app_id = $2
  )
  AND NOT EXISTS (
      SELECT 1 FROM channel_rollouts
      WHERE channel_rollouts.channel_id = channels.id
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
SELECT u.id, u.update_uuid, u.update_type, u.created_at, u.commit_hash, u.platform, u.message, u.checked_at, u.rollout_percentage, u.control_update_id
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
SELECT u.id, u.update_uuid, b.app_id, b.name AS branch_name, r.version AS runtime_version, u.update_type, u.commit_hash, u.message, u.platform, u.created_at, u.rollout_percentage, u.control_update_id
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

-- name: MarkUpdateAsChecked :execrows
-- Stamps the "complete and pickable" sentinel. The stamp is refused (0 rows) when it
-- would break a rollout invariant, which closes the publish/activation races
-- transactionally: a plain update cannot become visible while a rollout is active on
-- its (branch, rtv, platform), and a rollout update cannot activate once a newer
-- checked update superseded it during upload.
WITH target AS (
    SELECT u.id, u.branch_id, u.runtime_version_id, u.platform, u.rollout_percentage
    FROM updates u
    JOIN branches b ON u.branch_id = b.id
    WHERE u.id = $1 AND b.app_id = $2 AND b.name = $3
),
updated_rows AS (
    UPDATE updates
    SET checked_at = CURRENT_TIMESTAMP
    FROM target
    WHERE updates.id = target.id
      AND updates.branch_id = target.branch_id
      AND (
        (target.rollout_percentage IS NULL AND NOT EXISTS (
            SELECT 1 FROM updates a
            WHERE a.branch_id = target.branch_id
              AND a.runtime_version_id = target.runtime_version_id
              AND a.platform = target.platform
              AND a.rollout_percentage IS NOT NULL
              AND a.checked_at IS NOT NULL
        ))
        OR
        (target.rollout_percentage IS NOT NULL AND NOT EXISTS (
            SELECT 1 FROM updates n
            WHERE n.branch_id = target.branch_id
              AND n.runtime_version_id = target.runtime_version_id
              AND n.platform = target.platform
              AND n.checked_at IS NOT NULL
              AND n.id > target.id
        ))
      )
    RETURNING updates.runtime_version_id
)
UPDATE runtime_versions
SET updated_at = CURRENT_TIMESTAMP
WHERE id IN (SELECT runtime_version_id FROM updated_rows);

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
INSERT INTO users (id, email, password_hash, is_admin, enabled)
VALUES ($1, $2, $3, $4, $5)
RETURNING id, email, is_admin, enabled, created_at;

-- name: GetUserByEmail :one
SELECT * FROM users
WHERE email = $1 LIMIT 1;

-- name: GetUserByID :one
SELECT * FROM users
WHERE id = $1 LIMIT 1;

-- name: GetUsers :many
SELECT id, email, is_admin, enabled, created_at, last_connected_at FROM users
ORDER BY created_at ASC;

-- name: TouchUserLastConnectedAt :exec
UPDATE users
SET last_connected_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeleteUserByID :execresult
-- Locks the admin rows first so concurrent deletes/demotions/disables
-- serialize: deleting the last remaining admin matches no row instead of
-- leaving the dashboard without any admin. Disabled admins are excluded, since
-- an account that cannot sign in is no safety net.
WITH admins AS (
    SELECT id FROM users WHERE is_admin AND enabled ORDER BY id FOR UPDATE
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
    SELECT id FROM users WHERE is_admin AND enabled ORDER BY id FOR UPDATE
)
UPDATE users
SET is_admin = $2, updated_at = CURRENT_TIMESTAMP
WHERE users.id = $1
  AND ($2::boolean
       OR users.id NOT IN (SELECT id FROM admins)
       OR (SELECT COUNT(*) FROM admins) > 1);

-- name: UpdateUserEnabledByID :execresult
-- Same admin-row lock as DeleteUserByID: disabling the last remaining enabled
-- admin matches no row, so approving/revoking accounts can never lock the
-- dashboard out. Enabling ($2 true) always passes the guard but still takes
-- the lock, so it serializes with concurrent disables.
WITH admins AS (
    SELECT id FROM users WHERE is_admin AND enabled ORDER BY id FOR UPDATE
)
UPDATE users
SET enabled = $2, updated_at = CURRENT_TIMESTAMP
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
INSERT INTO sso_config (singleton, issuer, client_id, sealed_client_secret, provider_name, scopes, enabled, allowed_email_domains, allowed_groups, groups_claim, trust_unverified_email, manual_user_validation)
VALUES (TRUE, $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
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
    manual_user_validation = EXCLUDED.manual_user_validation,
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

-- The queries below back progressive rollouts (MIT core, control-plane mode only).

-- name: GetUpdateByUUID :one
-- App-scoped, checked-only lookup by the persistent update UUID. Backs the /assets
-- rollout fix: expo-updates sends Expo-Requested-Update-ID on every asset request, so
-- the exact update it is running can be served regardless of the rollout decision.
SELECT u.id, u.update_uuid, b.app_id, b.name AS branch_name, r.version AS runtime_version, u.update_type, u.commit_hash, u.message, u.platform, u.created_at, u.rollout_percentage, u.control_update_id
FROM updates u
INNER JOIN branches b ON u.branch_id = b.id
INNER JOIN runtime_versions r ON u.runtime_version_id = r.id
WHERE b.app_id = $1
  AND u.update_uuid = $2
  AND u.checked_at IS NOT NULL
LIMIT 1;

-- name: GetLatestUpdateWithRollout :one
-- Latest checked update for (branch, rtv, platform) plus its control, resolved through
-- the explicit control_update_id pointer (a LEFT JOIN on the composite PK, NOT a LIMIT-2
-- heuristic). Control fields are NULL when the update carries no rollout.
SELECT
    u.id,
    u.update_uuid,
    u.branch_id,
    u.runtime_version_id,
    u.update_type,
    u.commit_hash,
    u.message,
    u.platform,
    u.created_at,
    u.rollout_percentage,
    u.control_update_id,
    c.id AS control_id,
    c.created_at AS control_created_at,
    c.update_type AS control_update_type
FROM updates u
JOIN branches b ON u.branch_id = b.id
JOIN runtime_versions rv ON u.runtime_version_id = rv.id
LEFT JOIN updates c ON c.branch_id = u.branch_id AND c.id = u.control_update_id
WHERE b.app_id = $1
  AND b.name = $2
  AND rv.version = $3
  AND u.platform = $4
  AND u.checked_at IS NOT NULL
ORDER BY u.id DESC
LIMIT 1;

-- name: HasActiveRolloutUpdate :one
-- Fail-fast publish guard: reports whether (branch, rtv) already has an active
-- per-update rollout on any platform.
SELECT EXISTS (
    SELECT 1
    FROM updates u
    JOIN branches b ON u.branch_id = b.id
    JOIN runtime_versions rv ON u.runtime_version_id = rv.id
    WHERE b.app_id = $1
      AND b.name = $2
      AND rv.version = $3
      AND u.rollout_percentage IS NOT NULL
      AND u.checked_at IS NOT NULL
);

-- name: GetActiveRolloutUpdates :many
-- The active per-update rollout rows for (branch, rtv), one per platform.
SELECT u.id, u.platform, u.rollout_percentage, u.control_update_id, u.created_at
FROM updates u
JOIN branches b ON u.branch_id = b.id
JOIN runtime_versions rv ON u.runtime_version_id = rv.id
WHERE b.app_id = $1
  AND b.name = $2
  AND rv.version = $3
  AND u.rollout_percentage IS NOT NULL
  AND u.checked_at IS NOT NULL
ORDER BY u.platform ASC;

-- name: SetUpdateRolloutPercentage :execrows
-- Dashboard progression: sets the new percentage on every active rollout row for
-- (branch, rtv). The rollout_percentage < $4 guard enforces monotonic increase inside
-- the UPDATE itself so concurrent progressions cannot lower the percentage; the service
-- pre-reads only to produce a friendly 400. 0 rows means the rollout ended or was
-- progressed past $4 in a concurrent edit.
UPDATE updates
SET rollout_percentage = $4
WHERE branch_id = (SELECT branches.id FROM branches WHERE branches.app_id = $1 AND branches.name = $2)
  AND runtime_version_id = (SELECT runtime_versions.id FROM runtime_versions WHERE runtime_versions.app_id = $1 AND runtime_versions.version = $3)
  AND rollout_percentage IS NOT NULL
  AND rollout_percentage < $4
  AND checked_at IS NOT NULL;

-- name: ClearUpdateRollout :execrows
-- Ends the per-update rollout for (branch, rtv) by clearing the percentage on every
-- active row. Used by both "finish" (progress to 100) and "revert". control_update_id
-- is deliberately retained: it is the historical marker the dashboard uses to render
-- the finished-rollout state, and serving only ever reads it together with a non-NULL
-- rollout_percentage.
UPDATE updates
SET rollout_percentage = NULL
WHERE branch_id = (SELECT branches.id FROM branches WHERE branches.app_id = $1 AND branches.name = $2)
  AND runtime_version_id = (SELECT runtime_versions.id FROM runtime_versions WHERE runtime_versions.app_id = $1 AND runtime_versions.version = $3)
  AND rollout_percentage IS NOT NULL
  AND checked_at IS NOT NULL;

-- name: InsertUpdateWithRollout :one
-- Publishes an update carrying a rollout percentage. The resolved_control CTE resolves
-- the control (latest checked update of the same branch/rtv/platform) in the same
-- statement; control_id may be NULL for the first update of a branch.
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
),
resolved_control AS (
    SELECT u.id AS control_id
    FROM updates u
    WHERE u.branch_id = (SELECT resolved_branch_id FROM resolved_names)
      AND u.runtime_version_id = (SELECT resolved_runtime_version_id FROM resolved_names)
      AND u.platform = $6
      AND u.checked_at IS NOT NULL
    ORDER BY u.id DESC
    LIMIT 1
)
INSERT INTO updates (
    id,
    branch_id,
    runtime_version_id,
    update_type,
    platform,
    commit_hash,
    message,
    rollout_percentage,
    control_update_id
) VALUES (
    $1,
    (SELECT resolved_branch_id FROM resolved_names),
    (SELECT resolved_runtime_version_id FROM resolved_names),
    $5,
    $6,
    $7,
    $8,
    $9,
    (SELECT control_id FROM resolved_control)
)
RETURNING
    id,
    platform,
    commit_hash,
    message,
    created_at,
    rollout_percentage,
    control_update_id,
    (SELECT app_id FROM resolved_names) AS app_id,
    (SELECT branch_name FROM resolved_names) AS branch_name,
    (SELECT runtime_version FROM resolved_names) AS runtime_version;

-- name: InsertChannelRollout :execrows
-- App-scoped INSERT...SELECT that refuses an unmapped channel (branch_id IS NULL) and a
-- rollout branch equal to the channel's current default. 0 rows inserted => the service
-- disambiguates (404 unknown channel / 400 unmapped or same branch). 23505 on channel_id
-- => 409 already active.
INSERT INTO channel_rollouts (id, channel_id, rollout_branch_id, percentage)
SELECT $1, c.id, rb.id, $2
FROM channels c
JOIN branches rb ON rb.app_id = c.app_id AND rb.name = $5
WHERE c.app_id = $3
  AND c.name = $4
  AND c.branch_id IS NOT NULL
  AND rb.id <> c.branch_id;

-- name: GetChannelRollout :one
SELECT cr.id, cr.channel_id, ch.name AS channel_name,
    db.name AS default_branch_name,
    rb.name AS rollout_branch_name,
    cr.percentage, cr.created_at, cr.updated_at
FROM channel_rollouts cr
JOIN channels ch ON cr.channel_id = ch.id
JOIN branches db ON ch.branch_id = db.id
JOIN branches rb ON cr.rollout_branch_id = rb.id
WHERE ch.app_id = $1 AND ch.name = $2;

-- name: UpdateChannelRolloutPercentage :execrows
UPDATE channel_rollouts
SET percentage = $1, updated_at = CURRENT_TIMESTAMP
WHERE channel_id = (SELECT id FROM channels WHERE app_id = $2 AND name = $3);

-- name: DeleteChannelRollout :execrows
DELETE FROM channel_rollouts
WHERE channel_id = (SELECT id FROM channels WHERE app_id = $1 AND name = $2);

-- name: RepointChannelToRolloutBranch :execrows
-- Promote step: repoints the channel to its rollout branch. Runs with DeleteChannelRollout
-- inside a single transaction (Engine.WithTx). Not blocked by UpdateChannelBranchMapping's
-- rollout guard because it is a distinct statement.
UPDATE channels
SET branch_id = (
    SELECT rollout_branch_id FROM channel_rollouts WHERE channel_id = channels.id
)
WHERE app_id = $1 AND name = $2
  AND EXISTS (SELECT 1 FROM channel_rollouts WHERE channel_id = channels.id);

-- name: GetChannelRolloutsByBranch :many
-- Branch-delete guard: the channels whose active rollout serves this branch. FK RESTRICT
-- already blocks the delete; this yields the friendly channel list for the error message.
SELECT ch.name AS channel_name
FROM channel_rollouts cr
JOIN channels ch ON cr.channel_id = ch.id
WHERE cr.rollout_branch_id = (SELECT branches.id FROM branches WHERE branches.app_id = $1 AND branches.name = $2);
