-- name: ClaimDueJob :one
UPDATE jobs
SET
    claimed_at = now(),
    claimed_by = $1
WHERE id = (
    SELECT id
    FROM jobs
    WHERE enabled = true
      AND next_run_at <= now()
      AND (
            claimed_at IS NULL
            OR claimed_at < now() - ($2::interval)
          )
    ORDER BY next_run_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: ClaimManualTrigger :one
UPDATE manual_job_triggers
SET
    claimed_at = now(),
    claimed_by = $1
WHERE id = (
    SELECT id
    FROM manual_job_triggers
    WHERE completed_at IS NULL
      AND (
            claimed_at IS NULL
            OR claimed_at < now() - ($2::interval)
          )
    ORDER BY created_at ASC
    FOR UPDATE SKIP LOCKED
    LIMIT 1
)
RETURNING *;

-- name: GetManualTriggerJob :one
SELECT j.*
FROM manual_job_triggers t
JOIN jobs j ON j.id = t.job_id
WHERE t.id = $1;

-- name: CompleteManualTrigger :exec
UPDATE manual_job_triggers
SET
    completed_at = now(),
    claimed_at = NULL,
    claimed_by = NULL
WHERE id = $1;

-- name: ResetManualTriggerClaim :exec
UPDATE manual_job_triggers
SET
    claimed_at = NULL,
    claimed_by = NULL
WHERE id = $1;

-- name: CompleteScheduledJobExecution :exec
UPDATE jobs
SET
    last_run_at = $2,
    next_run_at = $3,
    claimed_at = NULL,
    claimed_by = NULL,
    updated_at = now()
WHERE id = $1;

-- name: CompleteManualJobExecution :exec
UPDATE jobs
SET
    last_run_at = $2,
    updated_at = now()
WHERE id = $1;

-- name: ReleaseClaim :exec
UPDATE jobs
SET
    claimed_at = NULL,
    claimed_by = NULL,
    updated_at = now()
WHERE id = $1;
