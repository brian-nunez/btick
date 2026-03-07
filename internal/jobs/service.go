package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/brian-nunez/go-echo-starter-template/internal/authorization"
	"github.com/brian-nunez/go-echo-starter-template/internal/db/sqlc"
	"github.com/brian-nunez/go-echo-starter-template/internal/scheduler"
	"github.com/google/uuid"
)

var (
	ErrNotFound         = errors.New("job not found")
	ErrValidationFailed = errors.New("job validation failed")
)

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type ValidationErrors []ValidationError

func (v ValidationErrors) Error() string {
	if len(v) == 0 {
		return "validation failed"
	}
	parts := make([]string, 0, len(v))
	for _, entry := range v {
		parts = append(parts, fmt.Sprintf("%s: %s", entry.Field, entry.Message))
	}
	return strings.Join(parts, ", ")
}

type CreateJobInput struct {
	Name           string            `json:"name"`
	Method         string            `json:"method"`
	URL            string            `json:"url"`
	Headers        map[string]string `json:"headers"`
	Body           json.RawMessage   `json:"body"`
	CronExpression string            `json:"cron_expression"`
	Timezone       string            `json:"timezone"`
	RetryMax       int32             `json:"retry_max"`
	TimeoutSeconds int32             `json:"timeout_seconds"`
	Enabled        bool              `json:"enabled"`
}

type PatchJobInput struct {
	Name           *string            `json:"name"`
	Method         *string            `json:"method"`
	URL            *string            `json:"url"`
	Headers        *map[string]string `json:"headers"`
	Body           *json.RawMessage   `json:"body"`
	CronExpression *string            `json:"cron_expression"`
	Timezone       *string            `json:"timezone"`
	RetryMax       *int32             `json:"retry_max"`
	TimeoutSeconds *int32             `json:"timeout_seconds"`
	Enabled        *bool              `json:"enabled"`
}

type Service struct {
	queries    *sqlc.Queries
	authorizer *authorization.Authorizer
}

func NewService(queries *sqlc.Queries, authorizer *authorization.Authorizer) *Service {
	return &Service{
		queries:    queries,
		authorizer: authorizer,
	}
}

func (s *Service) Create(ctx context.Context, principal *auth.Principal, input CreateJobInput) (sqlc.Job, error) {
	if err := s.authorizer.Require(principal, authorization.ActionJobsWrite, authorization.Resource{TenantID: principal.TenantID}); err != nil {
		return sqlc.Job{}, err
	}

	errs := validateCreateInput(input)
	if len(errs) > 0 {
		return sqlc.Job{}, errs
	}

	headersJSON, _ := json.Marshal(input.Headers)
	var bodyJSON []byte
	if len(input.Body) > 0 && string(input.Body) != "null" {
		bodyJSON = input.Body
	}

	var nextRunAt *time.Time
	if input.Enabled {
		nextRun, err := scheduler.NextRun(input.CronExpression, input.Timezone, time.Now().UTC())
		if err != nil {
			return sqlc.Job{}, ValidationErrors{{Field: "cron_expression", Message: err.Error()}}
		}
		nextRunAt = &nextRun
	}

	var createdBy *string
	if principal.SubjectID != "" {
		createdBy = &principal.SubjectID
	}

	job, err := s.queries.CreateJob(ctx, sqlc.CreateJobParams{
		ID:             uuid.New(),
		Name:           strings.TrimSpace(input.Name),
		Method:         strings.ToUpper(strings.TrimSpace(input.Method)),
		URL:            strings.TrimSpace(input.URL),
		Headers:        headersJSON,
		Body:           bodyJSON,
		CronExpression: strings.TrimSpace(input.CronExpression),
		Timezone:       strings.TrimSpace(input.Timezone),
		RetryMax:       input.RetryMax,
		TimeoutSeconds: input.TimeoutSeconds,
		Enabled:        input.Enabled,
		NextRunAt:      nextRunAt,
		TenantID:       principal.TenantID,
		CreatedBy:      createdBy,
	})
	if err != nil {
		return sqlc.Job{}, err
	}

	return job, nil
}

func (s *Service) List(ctx context.Context, principal *auth.Principal) ([]sqlc.Job, error) {
	if err := s.authorizer.Require(principal, authorization.ActionJobsRead, authorization.Resource{TenantID: principal.TenantID}); err != nil {
		return nil, err
	}

	jobs, err := s.queries.ListJobs(ctx)
	if err != nil {
		return nil, err
	}

	if principal.Kind == auth.PrincipalKindSystem && principal.TenantID == nil {
		return jobs, nil
	}

	filtered := make([]sqlc.Job, 0, len(jobs))
	for _, job := range jobs {
		if job.TenantID != nil && principal.TenantID != nil && *job.TenantID == *principal.TenantID {
			filtered = append(filtered, job)
		}
	}

	return filtered, nil
}

func (s *Service) Get(ctx context.Context, principal *auth.Principal, jobID uuid.UUID) (sqlc.Job, error) {
	job, err := s.queries.GetJob(ctx, jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.Job{}, ErrNotFound
		}
		return sqlc.Job{}, err
	}

	if err := s.authorizer.Require(principal, authorization.ActionJobsRead, authorization.Resource{TenantID: job.TenantID, OwnerID: job.CreatedBy}); err != nil {
		return sqlc.Job{}, err
	}

	return job, nil
}

func (s *Service) UpdatePatch(ctx context.Context, principal *auth.Principal, jobID uuid.UUID, patch PatchJobInput) (sqlc.Job, error) {
	currentJob, err := s.Get(ctx, principal, jobID)
	if err != nil {
		return sqlc.Job{}, err
	}

	if err := s.authorizer.Require(principal, authorization.ActionJobsWrite, authorization.Resource{TenantID: currentJob.TenantID, OwnerID: currentJob.CreatedBy}); err != nil {
		return sqlc.Job{}, err
	}

	input, err := mergePatch(currentJob, patch)
	if err != nil {
		return sqlc.Job{}, err
	}

	errs := validateCreateInput(input)
	if len(errs) > 0 {
		return sqlc.Job{}, errs
	}

	headersJSON, _ := json.Marshal(input.Headers)
	var bodyJSON []byte
	if len(input.Body) > 0 && string(input.Body) != "null" {
		bodyJSON = input.Body
	}

	var nextRunAt *time.Time
	if input.Enabled {
		nextRun, err := scheduler.NextRun(input.CronExpression, input.Timezone, time.Now().UTC())
		if err != nil {
			return sqlc.Job{}, ValidationErrors{{Field: "cron_expression", Message: err.Error()}}
		}
		nextRunAt = &nextRun
	}

	updated, err := s.queries.UpdateJob(ctx, sqlc.UpdateJobParams{
		ID:             currentJob.ID,
		Name:           strings.TrimSpace(input.Name),
		Method:         strings.ToUpper(strings.TrimSpace(input.Method)),
		URL:            strings.TrimSpace(input.URL),
		Headers:        headersJSON,
		Body:           bodyJSON,
		CronExpression: strings.TrimSpace(input.CronExpression),
		Timezone:       strings.TrimSpace(input.Timezone),
		RetryMax:       input.RetryMax,
		TimeoutSeconds: input.TimeoutSeconds,
		Enabled:        input.Enabled,
		NextRunAt:      nextRunAt,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.Job{}, ErrNotFound
		}
		return sqlc.Job{}, err
	}

	return updated, nil
}

func (s *Service) Delete(ctx context.Context, principal *auth.Principal, jobID uuid.UUID) error {
	job, err := s.Get(ctx, principal, jobID)
	if err != nil {
		return err
	}

	if err := s.authorizer.Require(principal, authorization.ActionJobsDelete, authorization.Resource{TenantID: job.TenantID, OwnerID: job.CreatedBy}); err != nil {
		return err
	}

	if err := s.queries.DeleteJob(ctx, job.ID); err != nil {
		return err
	}

	return nil
}

func (s *Service) Pause(ctx context.Context, principal *auth.Principal, jobID uuid.UUID) (sqlc.Job, error) {
	job, err := s.Get(ctx, principal, jobID)
	if err != nil {
		return sqlc.Job{}, err
	}

	if err := s.authorizer.Require(principal, authorization.ActionJobsWrite, authorization.Resource{TenantID: job.TenantID, OwnerID: job.CreatedBy}); err != nil {
		return sqlc.Job{}, err
	}

	paused, err := s.queries.PauseJob(ctx, job.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.Job{}, ErrNotFound
		}
		return sqlc.Job{}, err
	}

	return paused, nil
}

func (s *Service) Resume(ctx context.Context, principal *auth.Principal, jobID uuid.UUID) (sqlc.Job, error) {
	job, err := s.Get(ctx, principal, jobID)
	if err != nil {
		return sqlc.Job{}, err
	}

	if err := s.authorizer.Require(principal, authorization.ActionJobsWrite, authorization.Resource{TenantID: job.TenantID, OwnerID: job.CreatedBy}); err != nil {
		return sqlc.Job{}, err
	}

	nextRunAt, err := scheduler.NextRun(job.CronExpression, job.Timezone, time.Now().UTC())
	if err != nil {
		return sqlc.Job{}, err
	}

	resumed, err := s.queries.ResumeJob(ctx, job.ID, nextRunAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return sqlc.Job{}, ErrNotFound
		}
		return sqlc.Job{}, err
	}

	return resumed, nil
}

func (s *Service) Trigger(ctx context.Context, principal *auth.Principal, jobID uuid.UUID) error {
	job, err := s.Get(ctx, principal, jobID)
	if err != nil {
		return err
	}

	if err := s.authorizer.Require(principal, authorization.ActionJobsTrigger, authorization.Resource{TenantID: job.TenantID, OwnerID: job.CreatedBy}); err != nil {
		return err
	}

	triggeredBy := principal.SubjectID
	if _, err := s.queries.InsertManualTrigger(ctx, uuid.New(), job.ID, &triggeredBy); err != nil {
		return err
	}

	return nil
}

func validateCreateInput(input CreateJobInput) ValidationErrors {
	errors := ValidationErrors{}

	if strings.TrimSpace(input.Name) == "" {
		errors = append(errors, ValidationError{Field: "name", Message: "name is required"})
	}

	method := strings.ToUpper(strings.TrimSpace(input.Method))
	if method == "" {
		errors = append(errors, ValidationError{Field: "method", Message: "method is required"})
	} else if !validHTTPMethod(method) {
		errors = append(errors, ValidationError{Field: "method", Message: "method must be a valid HTTP method"})
	}

	if strings.TrimSpace(input.URL) == "" {
		errors = append(errors, ValidationError{Field: "url", Message: "url is required"})
	} else {
		parsedURL, err := url.ParseRequestURI(input.URL)
		if err != nil || parsedURL.Scheme == "" || parsedURL.Host == "" {
			errors = append(errors, ValidationError{Field: "url", Message: "url must include scheme and host"})
		}
	}

	if strings.TrimSpace(input.CronExpression) == "" {
		errors = append(errors, ValidationError{Field: "cron_expression", Message: "cron_expression is required"})
	} else if err := scheduler.ValidateCron(input.CronExpression); err != nil {
		errors = append(errors, ValidationError{Field: "cron_expression", Message: err.Error()})
	}

	if strings.TrimSpace(input.Timezone) == "" {
		errors = append(errors, ValidationError{Field: "timezone", Message: "timezone is required"})
	} else if err := scheduler.ValidateTimezone(input.Timezone); err != nil {
		errors = append(errors, ValidationError{Field: "timezone", Message: err.Error()})
	}

	if input.RetryMax < 0 {
		errors = append(errors, ValidationError{Field: "retry_max", Message: "retry_max must be >= 0"})
	}

	if input.TimeoutSeconds <= 0 {
		errors = append(errors, ValidationError{Field: "timeout_seconds", Message: "timeout_seconds must be > 0"})
	}

	if input.Headers == nil {
		errors = append(errors, ValidationError{Field: "headers", Message: "headers is required"})
	}

	if input.Headers != nil {
		for key := range input.Headers {
			if strings.TrimSpace(key) == "" {
				errors = append(errors, ValidationError{Field: "headers", Message: "headers keys must be non-empty"})
				break
			}
		}
	}

	if len(input.Body) > 0 && string(input.Body) != "null" {
		var raw any
		if err := json.Unmarshal(input.Body, &raw); err != nil {
			errors = append(errors, ValidationError{Field: "body", Message: "body must be valid JSON"})
		}
	}

	return errors
}

func mergePatch(current sqlc.Job, patch PatchJobInput) (CreateJobInput, error) {
	headers := map[string]string{}
	if len(current.Headers) > 0 {
		if err := json.Unmarshal(current.Headers, &headers); err != nil {
			return CreateJobInput{}, fmt.Errorf("decode headers: %w", err)
		}
	}

	input := CreateJobInput{
		Name:           current.Name,
		Method:         current.Method,
		URL:            current.URL,
		Headers:        headers,
		Body:           cloneJSON(current.Body),
		CronExpression: current.CronExpression,
		Timezone:       current.Timezone,
		RetryMax:       current.RetryMax,
		TimeoutSeconds: current.TimeoutSeconds,
		Enabled:        current.Enabled,
	}

	if patch.Name != nil {
		input.Name = *patch.Name
	}
	if patch.Method != nil {
		input.Method = *patch.Method
	}
	if patch.URL != nil {
		input.URL = *patch.URL
	}
	if patch.Headers != nil {
		input.Headers = *patch.Headers
	}
	if patch.Body != nil {
		input.Body = cloneJSON(*patch.Body)
	}
	if patch.CronExpression != nil {
		input.CronExpression = *patch.CronExpression
	}
	if patch.Timezone != nil {
		input.Timezone = *patch.Timezone
	}
	if patch.RetryMax != nil {
		input.RetryMax = *patch.RetryMax
	}
	if patch.TimeoutSeconds != nil {
		input.TimeoutSeconds = *patch.TimeoutSeconds
	}
	if patch.Enabled != nil {
		input.Enabled = *patch.Enabled
	}

	return input, nil
}

func validHTTPMethod(method string) bool {
	switch method {
	case http.MethodGet,
		http.MethodPost,
		http.MethodPut,
		http.MethodPatch,
		http.MethodDelete,
		http.MethodHead,
		http.MethodOptions,
		http.MethodTrace,
		http.MethodConnect:
		return true
	default:
		return false
	}
}

func cloneJSON(value []byte) []byte {
	if len(value) == 0 {
		return nil
	}
	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}
