-- name: CreateAPIKey :one
INSERT INTO api_keys (
    id,
    name,
    key_prefix,
    key_hash,
    scopes,
    tenant_id,
    created_by,
    expires_at
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING *;

-- name: ListAPIKeys :many
SELECT *
FROM api_keys
ORDER BY created_at DESC;

-- name: GetAPIKeyByPrefix :one
SELECT *
FROM api_keys
WHERE key_prefix = $1;

-- name: GetAPIKey :one
SELECT *
FROM api_keys
WHERE id = $1;

-- name: RevokeAPIKey :one
UPDATE api_keys
SET revoked_at = now()
WHERE id = $1
RETURNING *;

-- name: UpdateAPIKeyLastUsedAt :exec
UPDATE api_keys
SET last_used_at = now()
WHERE id = $1;
