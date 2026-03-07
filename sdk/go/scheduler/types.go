package scheduler

import "time"

type Job struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	Body           any               `json:"body"`
	CronExpression string            `json:"cron_expression"`
	Timezone       string            `json:"timezone"`
	RetryMax       int32             `json:"retry_max"`
	TimeoutSeconds int32             `json:"timeout_seconds"`
	Enabled        bool              `json:"enabled"`
	NextRunAt      *time.Time        `json:"next_run_at"`
	LastRunAt      *time.Time        `json:"last_run_at"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	TenantID       *string           `json:"tenant_id"`
	CreatedBy      *string           `json:"created_by"`
}

type CreateJobRequest struct {
	Name           string            `json:"name"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	Body           any               `json:"body,omitempty"`
	CronExpression string            `json:"cron_expression"`
	Timezone       string            `json:"timezone"`
	RetryMax       int32             `json:"retry_max"`
	TimeoutSeconds int32             `json:"timeout_seconds"`
	Enabled        bool              `json:"enabled"`
}

type UpdateJobRequest struct {
	Name           *string            `json:"name,omitempty"`
	Method         *string            `json:"method,omitempty"`
	URL            *string            `json:"url,omitempty"`
	Headers        *map[string]string `json:"headers,omitempty"`
	Body           any                `json:"body,omitempty"`
	CronExpression *string            `json:"cron_expression,omitempty"`
	Timezone       *string            `json:"timezone,omitempty"`
	RetryMax       *int32             `json:"retry_max,omitempty"`
	TimeoutSeconds *int32             `json:"timeout_seconds,omitempty"`
	Enabled        *bool              `json:"enabled,omitempty"`
}

type Run struct {
	ID                 string              `json:"id"`
	JobID              string              `json:"job_id"`
	TriggerType        string              `json:"trigger_type"`
	AttemptNumber      int32               `json:"attempt_number"`
	Status             string              `json:"status"`
	RequestMethod      string              `json:"request_method"`
	RequestURL         string              `json:"request_url"`
	RequestHeaders     map[string]string   `json:"request_headers"`
	RequestBody        any                 `json:"request_body"`
	ResponseStatusCode *int32              `json:"response_status_code"`
	ResponseHeaders    map[string][]string `json:"response_headers"`
	ResponseBody       *string             `json:"response_body"`
	ErrorMessage       *string             `json:"error_message"`
	StartedAt          time.Time           `json:"started_at"`
	FinishedAt         *time.Time          `json:"finished_at"`
}

type APIKey struct {
	ID         string     `json:"id"`
	Name       string     `json:"name"`
	KeyPrefix  string     `json:"key_prefix"`
	Scopes     []string   `json:"scopes"`
	TenantID   *string    `json:"tenant_id"`
	CreatedBy  *string    `json:"created_by"`
	LastUsedAt *time.Time `json:"last_used_at"`
	ExpiresAt  *time.Time `json:"expires_at"`
	RevokedAt  *time.Time `json:"revoked_at"`
	CreatedAt  time.Time  `json:"created_at"`
}

type CreateAPIKeyRequest struct {
	Name      string     `json:"name"`
	Scopes    []string   `json:"scopes"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

type CreateAPIKeyResponse struct {
	APIKey APIKey `json:"api_key"`
	RawKey string `json:"raw_key"`
}
