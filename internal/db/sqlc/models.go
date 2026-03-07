package sqlc

import (
	"time"

	"github.com/google/uuid"
)

type Job struct {
	ID             uuid.UUID  `json:"id"`
	Name           string     `json:"name"`
	Method         string     `json:"method"`
	URL            string     `json:"url"`
	Headers        []byte     `json:"headers"`
	Body           []byte     `json:"body,omitempty"`
	CronExpression string     `json:"cron_expression"`
	Timezone       string     `json:"timezone"`
	RetryMax       int32      `json:"retry_max"`
	TimeoutSeconds int32      `json:"timeout_seconds"`
	Enabled        bool       `json:"enabled"`
	NextRunAt      *time.Time `json:"next_run_at,omitempty"`
	LastRunAt      *time.Time `json:"last_run_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	ClaimedAt      *time.Time `json:"claimed_at,omitempty"`
	ClaimedBy      *string    `json:"claimed_by,omitempty"`
	TenantID       *uuid.UUID `json:"tenant_id,omitempty"`
	CreatedBy      *string    `json:"created_by,omitempty"`
}

type JobRun struct {
	ID                 uuid.UUID  `json:"id"`
	JobID              uuid.UUID  `json:"job_id"`
	TriggerType        string     `json:"trigger_type"`
	AttemptNumber      int32      `json:"attempt_number"`
	Status             string     `json:"status"`
	RequestMethod      string     `json:"request_method"`
	RequestURL         string     `json:"request_url"`
	RequestHeaders     []byte     `json:"request_headers"`
	RequestBody        []byte     `json:"request_body,omitempty"`
	ResponseStatusCode *int32     `json:"response_status_code,omitempty"`
	ResponseHeaders    []byte     `json:"response_headers,omitempty"`
	ResponseBody       *string    `json:"response_body,omitempty"`
	ErrorMessage       *string    `json:"error_message,omitempty"`
	StartedAt          time.Time  `json:"started_at"`
	FinishedAt         *time.Time `json:"finished_at,omitempty"`
}

type APIKey struct {
	ID        uuid.UUID  `json:"id"`
	Name      string     `json:"name"`
	KeyPrefix string     `json:"key_prefix"`
	KeyHash   string     `json:"-"`
	Scopes    []byte     `json:"scopes"`
	TenantID  *uuid.UUID `json:"tenant_id,omitempty"`
	CreatedBy *string    `json:"created_by,omitempty"`

	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type ManualJobTrigger struct {
	ID          uuid.UUID  `json:"id"`
	JobID       uuid.UUID  `json:"job_id"`
	TriggeredBy *string    `json:"triggered_by,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	ClaimedAt   *time.Time `json:"claimed_at,omitempty"`
	ClaimedBy   *string    `json:"claimed_by,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	Roles        []byte     `json:"roles"`
	Scopes       []byte     `json:"scopes"`
	TenantID     *uuid.UUID `json:"tenant_id,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}
