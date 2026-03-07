package sqlc

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type DBTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type Queries struct {
	db DBTX
}

func New(db DBTX) *Queries {
	return &Queries{db: db}
}

type CreateJobParams struct {
	ID             uuid.UUID
	Name           string
	Method         string
	URL            string
	Headers        []byte
	Body           []byte
	CronExpression string
	Timezone       string
	RetryMax       int32
	TimeoutSeconds int32
	Enabled        bool
	NextRunAt      *time.Time
	TenantID       *uuid.UUID
	CreatedBy      *string
}

func (q *Queries) CreateJob(ctx context.Context, arg CreateJobParams) (Job, error) {
	row := q.db.QueryRowContext(ctx, `
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
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)
		RETURNING id, name, method, url, headers, body, cron_expression, timezone, retry_max,
			timeout_seconds, enabled, next_run_at, last_run_at, created_at, updated_at,
			claimed_at, claimed_by, tenant_id, created_by
	`,
		arg.ID,
		arg.Name,
		arg.Method,
		arg.URL,
		arg.Headers,
		nullableBytes(arg.Body),
		arg.CronExpression,
		arg.Timezone,
		arg.RetryMax,
		arg.TimeoutSeconds,
		arg.Enabled,
		nullableTime(arg.NextRunAt),
		nullableUUID(arg.TenantID),
		nullableString(arg.CreatedBy),
	)
	return scanJob(row)
}

func (q *Queries) ListJobs(ctx context.Context) ([]Job, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, name, method, url, headers, body, cron_expression, timezone, retry_max,
			timeout_seconds, enabled, next_run_at, last_run_at, created_at, updated_at,
			claimed_at, claimed_by, tenant_id, created_by
		FROM jobs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanJobs(rows)
}

func (q *Queries) GetJob(ctx context.Context, id uuid.UUID) (Job, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, name, method, url, headers, body, cron_expression, timezone, retry_max,
			timeout_seconds, enabled, next_run_at, last_run_at, created_at, updated_at,
			claimed_at, claimed_by, tenant_id, created_by
		FROM jobs
		WHERE id = $1
	`, id)
	return scanJob(row)
}

type UpdateJobParams struct {
	ID             uuid.UUID
	Name           string
	Method         string
	URL            string
	Headers        []byte
	Body           []byte
	CronExpression string
	Timezone       string
	RetryMax       int32
	TimeoutSeconds int32
	Enabled        bool
	NextRunAt      *time.Time
}

func (q *Queries) UpdateJob(ctx context.Context, arg UpdateJobParams) (Job, error) {
	row := q.db.QueryRowContext(ctx, `
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
		RETURNING id, name, method, url, headers, body, cron_expression, timezone, retry_max,
			timeout_seconds, enabled, next_run_at, last_run_at, created_at, updated_at,
			claimed_at, claimed_by, tenant_id, created_by
	`,
		arg.ID,
		arg.Name,
		arg.Method,
		arg.URL,
		arg.Headers,
		nullableBytes(arg.Body),
		arg.CronExpression,
		arg.Timezone,
		arg.RetryMax,
		arg.TimeoutSeconds,
		arg.Enabled,
		nullableTime(arg.NextRunAt),
	)
	return scanJob(row)
}

func (q *Queries) DeleteJob(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM jobs WHERE id = $1`, id)
	return err
}

func (q *Queries) PauseJob(ctx context.Context, id uuid.UUID) (Job, error) {
	row := q.db.QueryRowContext(ctx, `
		UPDATE jobs
		SET
			enabled = false,
			updated_at = now(),
			claimed_at = NULL,
			claimed_by = NULL
		WHERE id = $1
		RETURNING id, name, method, url, headers, body, cron_expression, timezone, retry_max,
			timeout_seconds, enabled, next_run_at, last_run_at, created_at, updated_at,
			claimed_at, claimed_by, tenant_id, created_by
	`, id)
	return scanJob(row)
}

func (q *Queries) ResumeJob(ctx context.Context, id uuid.UUID, nextRunAt time.Time) (Job, error) {
	row := q.db.QueryRowContext(ctx, `
		UPDATE jobs
		SET
			enabled = true,
			next_run_at = $2,
			updated_at = now()
		WHERE id = $1
		RETURNING id, name, method, url, headers, body, cron_expression, timezone, retry_max,
			timeout_seconds, enabled, next_run_at, last_run_at, created_at, updated_at,
			claimed_at, claimed_by, tenant_id, created_by
	`, id, nextRunAt)
	return scanJob(row)
}

func (q *Queries) ClearJobClaim(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE jobs
		SET
			claimed_at = NULL,
			claimed_by = NULL,
			updated_at = now()
		WHERE id = $1
	`, id)
	return err
}

func (q *Queries) InsertManualTrigger(ctx context.Context, id uuid.UUID, jobID uuid.UUID, triggeredBy *string) (ManualJobTrigger, error) {
	row := q.db.QueryRowContext(ctx, `
		INSERT INTO manual_job_triggers (
			id,
			job_id,
			triggered_by
		)
		VALUES ($1, $2, $3)
		RETURNING id, job_id, triggered_by, created_at, claimed_at, claimed_by, completed_at
	`, id, jobID, nullableString(triggeredBy))
	return scanManualTrigger(row)
}

type CreateJobRunParams struct {
	ID             uuid.UUID
	JobID          uuid.UUID
	TriggerType    string
	AttemptNumber  int32
	Status         string
	RequestMethod  string
	RequestURL     string
	RequestHeaders []byte
	RequestBody    []byte
	StartedAt      time.Time
}

func (q *Queries) CreateJobRun(ctx context.Context, arg CreateJobRunParams) (JobRun, error) {
	row := q.db.QueryRowContext(ctx, `
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
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
		RETURNING id, job_id, trigger_type, attempt_number, status, request_method, request_url,
			request_headers, request_body, response_status_code, response_headers, response_body,
			error_message, started_at, finished_at
	`,
		arg.ID,
		arg.JobID,
		arg.TriggerType,
		arg.AttemptNumber,
		arg.Status,
		arg.RequestMethod,
		arg.RequestURL,
		arg.RequestHeaders,
		nullableBytes(arg.RequestBody),
		arg.StartedAt,
	)
	return scanJobRun(row)
}

func (q *Queries) UpdateRunningAttempt(ctx context.Context, runID uuid.UUID, attemptNumber int32) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE job_runs
		SET
			attempt_number = $2,
			status = 'running'
		WHERE id = $1
	`, runID, attemptNumber)
	return err
}

type CompleteRunSuccessParams struct {
	RunID              uuid.UUID
	AttemptNumber      int32
	ResponseStatusCode int32
	ResponseHeaders    []byte
	ResponseBody       string
	FinishedAt         time.Time
}

func (q *Queries) CompleteRunSuccess(ctx context.Context, arg CompleteRunSuccessParams) (JobRun, error) {
	row := q.db.QueryRowContext(ctx, `
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
		RETURNING id, job_id, trigger_type, attempt_number, status, request_method, request_url,
			request_headers, request_body, response_status_code, response_headers, response_body,
			error_message, started_at, finished_at
	`, arg.RunID, arg.AttemptNumber, arg.ResponseStatusCode, nullableBytes(arg.ResponseHeaders), arg.ResponseBody, arg.FinishedAt)
	return scanJobRun(row)
}

type CompleteRunFailureParams struct {
	RunID              uuid.UUID
	AttemptNumber      int32
	ResponseStatusCode *int32
	ResponseHeaders    []byte
	ResponseBody       *string
	ErrorMessage       string
	FinishedAt         time.Time
}

func (q *Queries) CompleteRunFailure(ctx context.Context, arg CompleteRunFailureParams) (JobRun, error) {
	row := q.db.QueryRowContext(ctx, `
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
		RETURNING id, job_id, trigger_type, attempt_number, status, request_method, request_url,
			request_headers, request_body, response_status_code, response_headers, response_body,
			error_message, started_at, finished_at
	`,
		arg.RunID,
		arg.AttemptNumber,
		nullableInt32(arg.ResponseStatusCode),
		nullableBytes(arg.ResponseHeaders),
		nullableString(arg.ResponseBody),
		arg.ErrorMessage,
		arg.FinishedAt,
	)
	return scanJobRun(row)
}

func (q *Queries) ListRunsByJob(ctx context.Context, jobID uuid.UUID, limit int32) ([]JobRun, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, job_id, trigger_type, attempt_number, status, request_method, request_url,
			request_headers, request_body, response_status_code, response_headers, response_body,
			error_message, started_at, finished_at
		FROM job_runs
		WHERE job_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`, jobID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	runs := make([]JobRun, 0)
	for rows.Next() {
		run, err := scanJobRun(rows)
		if err != nil {
			return nil, err
		}
		runs = append(runs, run)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return runs, nil
}

func (q *Queries) GetRun(ctx context.Context, id uuid.UUID) (JobRun, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, job_id, trigger_type, attempt_number, status, request_method, request_url,
			request_headers, request_body, response_status_code, response_headers, response_body,
			error_message, started_at, finished_at
		FROM job_runs
		WHERE id = $1
	`, id)
	return scanJobRun(row)
}

type CreateAPIKeyParams struct {
	ID        uuid.UUID
	Name      string
	KeyPrefix string
	KeyHash   string
	Scopes    []byte
	TenantID  *uuid.UUID
	CreatedBy *string
	ExpiresAt *time.Time
}

func (q *Queries) CreateAPIKey(ctx context.Context, arg CreateAPIKeyParams) (APIKey, error) {
	row := q.db.QueryRowContext(ctx, `
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
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)
		RETURNING id, name, key_prefix, key_hash, scopes, tenant_id, created_by,
			last_used_at, expires_at, revoked_at, created_at
	`,
		arg.ID,
		arg.Name,
		arg.KeyPrefix,
		arg.KeyHash,
		arg.Scopes,
		nullableUUID(arg.TenantID),
		nullableString(arg.CreatedBy),
		nullableTime(arg.ExpiresAt),
	)
	return scanAPIKey(row)
}

func (q *Queries) ListAPIKeys(ctx context.Context) ([]APIKey, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT id, name, key_prefix, key_hash, scopes, tenant_id, created_by,
			last_used_at, expires_at, revoked_at, created_at
		FROM api_keys
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	keys := make([]APIKey, 0)
	for rows.Next() {
		key, err := scanAPIKey(rows)
		if err != nil {
			return nil, err
		}
		keys = append(keys, key)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return keys, nil
}

func (q *Queries) GetAPIKey(ctx context.Context, id uuid.UUID) (APIKey, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, name, key_prefix, key_hash, scopes, tenant_id, created_by,
			last_used_at, expires_at, revoked_at, created_at
		FROM api_keys
		WHERE id = $1
	`, id)
	return scanAPIKey(row)
}

func (q *Queries) GetAPIKeyByPrefix(ctx context.Context, prefix string) (APIKey, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, name, key_prefix, key_hash, scopes, tenant_id, created_by,
			last_used_at, expires_at, revoked_at, created_at
		FROM api_keys
		WHERE key_prefix = $1
	`, prefix)
	return scanAPIKey(row)
}

func (q *Queries) RevokeAPIKey(ctx context.Context, id uuid.UUID) (APIKey, error) {
	row := q.db.QueryRowContext(ctx, `
		UPDATE api_keys
		SET revoked_at = now()
		WHERE id = $1
		RETURNING id, name, key_prefix, key_hash, scopes, tenant_id, created_by,
			last_used_at, expires_at, revoked_at, created_at
	`, id)
	return scanAPIKey(row)
}

func (q *Queries) UpdateAPIKeyLastUsedAt(ctx context.Context, id uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE api_keys
		SET last_used_at = now()
		WHERE id = $1
	`, id)
	return err
}

type CreateUserParams struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Roles        []byte
	Scopes       []byte
	TenantID     *uuid.UUID
}

func (q *Queries) CreateUser(ctx context.Context, arg CreateUserParams) (User, error) {
	row := q.db.QueryRowContext(ctx, `
		INSERT INTO users (
			id,
			email,
			password_hash,
			roles,
			scopes,
			tenant_id
		)
		VALUES ($1,$2,$3,$4,$5,$6)
		RETURNING id, email, password_hash, roles, scopes, tenant_id, created_at, updated_at
	`,
		arg.ID,
		arg.Email,
		arg.PasswordHash,
		arg.Roles,
		arg.Scopes,
		nullableUUID(arg.TenantID),
	)
	return scanUser(row)
}

func (q *Queries) GetUserByEmail(ctx context.Context, email string) (User, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, roles, scopes, tenant_id, created_at, updated_at
		FROM users
		WHERE email = $1
	`, email)
	return scanUser(row)
}

func (q *Queries) ClaimDueJob(ctx context.Context, workerID string, staleInterval string) (*Job, error) {
	row := q.db.QueryRowContext(ctx, `
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
		RETURNING id, name, method, url, headers, body, cron_expression, timezone, retry_max,
			timeout_seconds, enabled, next_run_at, last_run_at, created_at, updated_at,
			claimed_at, claimed_by, tenant_id, created_by
	`, workerID, staleInterval)
	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &job, nil
}

func (q *Queries) ClaimManualTrigger(ctx context.Context, workerID string, staleInterval string) (*ManualJobTrigger, error) {
	row := q.db.QueryRowContext(ctx, `
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
		RETURNING id, job_id, triggered_by, created_at, claimed_at, claimed_by, completed_at
	`, workerID, staleInterval)
	trigger, err := scanManualTrigger(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &trigger, nil
}

func (q *Queries) GetManualTriggerJob(ctx context.Context, triggerID uuid.UUID) (Job, error) {
	row := q.db.QueryRowContext(ctx, `
		SELECT j.id, j.name, j.method, j.url, j.headers, j.body, j.cron_expression, j.timezone, j.retry_max,
			j.timeout_seconds, j.enabled, j.next_run_at, j.last_run_at, j.created_at, j.updated_at,
			j.claimed_at, j.claimed_by, j.tenant_id, j.created_by
		FROM manual_job_triggers t
		JOIN jobs j ON j.id = t.job_id
		WHERE t.id = $1
	`, triggerID)
	return scanJob(row)
}

func (q *Queries) CompleteManualTrigger(ctx context.Context, triggerID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE manual_job_triggers
		SET
			completed_at = now(),
			claimed_at = NULL,
			claimed_by = NULL
		WHERE id = $1
	`, triggerID)
	return err
}

func (q *Queries) ResetManualTriggerClaim(ctx context.Context, triggerID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE manual_job_triggers
		SET
			claimed_at = NULL,
			claimed_by = NULL
		WHERE id = $1
	`, triggerID)
	return err
}

func (q *Queries) CompleteScheduledJobExecution(ctx context.Context, jobID uuid.UUID, lastRunAt time.Time, nextRunAt time.Time) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE jobs
		SET
			last_run_at = $2,
			next_run_at = $3,
			claimed_at = NULL,
			claimed_by = NULL,
			updated_at = now()
		WHERE id = $1
	`, jobID, lastRunAt, nextRunAt)
	return err
}

func (q *Queries) CompleteManualJobExecution(ctx context.Context, jobID uuid.UUID, lastRunAt time.Time) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE jobs
		SET
			last_run_at = $2,
			updated_at = now()
		WHERE id = $1
	`, jobID, lastRunAt)
	return err
}

func (q *Queries) ReleaseClaim(ctx context.Context, jobID uuid.UUID) error {
	_, err := q.db.ExecContext(ctx, `
		UPDATE jobs
		SET
			claimed_at = NULL,
			claimed_by = NULL,
			updated_at = now()
		WHERE id = $1
	`, jobID)
	return err
}

func scanJobs(rows *sql.Rows) ([]Job, error) {
	items := make([]Job, 0)
	for rows.Next() {
		item, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanJob(row scanner) (Job, error) {
	var item Job
	var body []byte
	var nextRun sql.NullTime
	var lastRun sql.NullTime
	var claimedAt sql.NullTime
	var claimedBy sql.NullString
	var tenantID sql.NullString
	var createdBy sql.NullString

	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.Method,
		&item.URL,
		&item.Headers,
		&body,
		&item.CronExpression,
		&item.Timezone,
		&item.RetryMax,
		&item.TimeoutSeconds,
		&item.Enabled,
		&nextRun,
		&lastRun,
		&item.CreatedAt,
		&item.UpdatedAt,
		&claimedAt,
		&claimedBy,
		&tenantID,
		&createdBy,
	)
	if err != nil {
		return Job{}, err
	}

	item.Body = cloneBytes(body)
	item.NextRunAt = nullTimePtr(nextRun)
	item.LastRunAt = nullTimePtr(lastRun)
	item.ClaimedAt = nullTimePtr(claimedAt)
	item.ClaimedBy = nullStringPtr(claimedBy)
	item.TenantID = nullUUIDPtr(tenantID)
	item.CreatedBy = nullStringPtr(createdBy)

	return item, nil
}

func scanJobRun(row scanner) (JobRun, error) {
	var item JobRun
	var requestBody []byte
	var responseStatus sql.NullInt32
	var responseHeaders []byte
	var responseBody sql.NullString
	var errorMessage sql.NullString
	var finishedAt sql.NullTime

	err := row.Scan(
		&item.ID,
		&item.JobID,
		&item.TriggerType,
		&item.AttemptNumber,
		&item.Status,
		&item.RequestMethod,
		&item.RequestURL,
		&item.RequestHeaders,
		&requestBody,
		&responseStatus,
		&responseHeaders,
		&responseBody,
		&errorMessage,
		&item.StartedAt,
		&finishedAt,
	)
	if err != nil {
		return JobRun{}, err
	}

	item.RequestBody = cloneBytes(requestBody)
	if responseStatus.Valid {
		value := responseStatus.Int32
		item.ResponseStatusCode = &value
	}
	item.ResponseHeaders = cloneBytes(responseHeaders)
	item.ResponseBody = nullStringPtr(responseBody)
	item.ErrorMessage = nullStringPtr(errorMessage)
	item.FinishedAt = nullTimePtr(finishedAt)

	return item, nil
}

func scanAPIKey(row scanner) (APIKey, error) {
	var item APIKey
	var tenantID sql.NullString
	var createdBy sql.NullString
	var lastUsedAt sql.NullTime
	var expiresAt sql.NullTime
	var revokedAt sql.NullTime

	err := row.Scan(
		&item.ID,
		&item.Name,
		&item.KeyPrefix,
		&item.KeyHash,
		&item.Scopes,
		&tenantID,
		&createdBy,
		&lastUsedAt,
		&expiresAt,
		&revokedAt,
		&item.CreatedAt,
	)
	if err != nil {
		return APIKey{}, err
	}

	item.TenantID = nullUUIDPtr(tenantID)
	item.CreatedBy = nullStringPtr(createdBy)
	item.LastUsedAt = nullTimePtr(lastUsedAt)
	item.ExpiresAt = nullTimePtr(expiresAt)
	item.RevokedAt = nullTimePtr(revokedAt)

	return item, nil
}

func scanManualTrigger(row scanner) (ManualJobTrigger, error) {
	var item ManualJobTrigger
	var triggeredBy sql.NullString
	var claimedAt sql.NullTime
	var claimedBy sql.NullString
	var completedAt sql.NullTime

	err := row.Scan(
		&item.ID,
		&item.JobID,
		&triggeredBy,
		&item.CreatedAt,
		&claimedAt,
		&claimedBy,
		&completedAt,
	)
	if err != nil {
		return ManualJobTrigger{}, err
	}

	item.TriggeredBy = nullStringPtr(triggeredBy)
	item.ClaimedAt = nullTimePtr(claimedAt)
	item.ClaimedBy = nullStringPtr(claimedBy)
	item.CompletedAt = nullTimePtr(completedAt)

	return item, nil
}

func scanUser(row scanner) (User, error) {
	var item User
	var tenantID sql.NullString

	err := row.Scan(
		&item.ID,
		&item.Email,
		&item.PasswordHash,
		&item.Roles,
		&item.Scopes,
		&tenantID,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		return User{}, err
	}

	item.TenantID = nullUUIDPtr(tenantID)
	return item, nil
}

func nullTimePtr(v sql.NullTime) *time.Time {
	if !v.Valid {
		return nil
	}
	value := v.Time
	return &value
}

func nullStringPtr(v sql.NullString) *string {
	if !v.Valid {
		return nil
	}
	value := v.String
	return &value
}

func nullUUIDPtr(v sql.NullString) *uuid.UUID {
	if !v.Valid || v.String == "" {
		return nil
	}
	parsed, err := uuid.Parse(v.String)
	if err != nil {
		return nil
	}
	return &parsed
}

func nullableUUID(value *uuid.UUID) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableBytes(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableInt32(value *int32) any {
	if value == nil {
		return nil
	}
	return *value
}

func cloneBytes(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	out := make([]byte, len(value))
	copy(out, value)
	return out
}

func MustUUIDFromString(value string) *uuid.UUID {
	parsed, err := uuid.Parse(value)
	if err != nil {
		panic(fmt.Sprintf("invalid uuid %q: %v", value, err))
	}
	return &parsed
}
