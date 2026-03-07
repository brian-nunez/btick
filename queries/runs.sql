-- name: CreateJobRun :one
INSERT INTO job_runs (
    id,
    job_id,
    trigger_type,
    attempt_number,
    status,
    request_method,
    request_url,
    request_headers,
    request_body,
    started_at
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
    $10
)
RETURNING *;

-- name: UpdateRunningAttempt :exec
UPDATE job_runs
SET
    attempt_number = $2,
    status = 'running'
WHERE id = $1;

-- name: CompleteRunSuccess :one
UPDATE job_runs
SET
    status = 'success',
    attempt_number = $2,
    response_status_code = $3,
    response_headers = $4,
    response_body = $5,
    error_message = NULL,
    finished_at = $6
WHERE id = $1
RETURNING *;

-- name: CompleteRunFailure :one
UPDATE job_runs
SET
    status = 'failed',
    attempt_number = $2,
    response_status_code = $3,
    response_headers = $4,
    response_body = $5,
    error_message = $6,
    finished_at = $7
WHERE id = $1
RETURNING *;

-- name: ListRunsByJob :many
SELECT *
FROM job_runs
WHERE job_id = $1
ORDER BY started_at DESC
LIMIT $2;

-- name: GetRun :one
SELECT *
FROM job_runs
WHERE id = $1;
