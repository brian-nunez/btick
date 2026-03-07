-- name: CreateUser :one
INSERT INTO users (
    id,
    email,
    password_hash,
    roles,
    scopes,
    tenant_id
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING *;

-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = $1;
