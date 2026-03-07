-- name: CreateJob :one
INSERT INTO jobs (
    id,
    name,
    method,
    url,
    headers,
    body,
    cron_expression,
    timezone,
    retry_max,
    timeout_seconds,
    enabled,
    next_run_at,
    tenant_id,
    created_by
)
VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8,
    $9,
    $10,
    $11,
    $12,
    $13,
    $14
)
RETURNING *;

-- name: ListJobs :many
SELECT *
FROM jobs
ORDER BY created_at DESC;

-- name: GetJob :one
SELECT *
FROM jobs
WHERE id = $1;

-- name: UpdateJob :one
UPDATE jobs
SET
    name = $2,
    method = $3,
    url = $4,
    headers = $5,
    body = $6,
    cron_expression = $7,
    timezone = $8,
    retry_max = $9,
    timeout_seconds = $10,
    enabled = $11,
    next_run_at = $12,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteJob :exec
DELETE FROM jobs
WHERE id = $1;

-- name: PauseJob :one
UPDATE jobs
SET
    enabled = false,
    updated_at = now(),
    claimed_at = NULL,
    claimed_by = NULL
WHERE id = $1
RETURNING *;

-- name: ResumeJob :one
UPDATE jobs
SET
    enabled = true,
    next_run_at = $2,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: ClearJobClaim :exec
UPDATE jobs
SET
    claimed_at = NULL,
    claimed_by = NULL,
    updated_at = now()
WHERE id = $1;

-- name: InsertManualTrigger :one
INSERT INTO manual_job_triggers (
    id,
    job_id,
    triggered_by
)
VALUES (
    $1,
    $2,
    $3
)
RETURNING *;
