package worker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/brian-nunez/go-echo-starter-template/internal/db/sqlc"
	"github.com/brian-nunez/go-echo-starter-template/internal/scheduler"
	taskworker "github.com/brian-nunez/task-orchestration"
	"github.com/google/uuid"
)

type TriggerType string

const (
	TriggerTypeScheduled TriggerType = "scheduled"
	TriggerTypeManual    TriggerType = "manual"
)

type ExecutionTaskConfig struct {
	Queries         *sqlc.Queries
	Logger          *log.Logger
	HTTPClient      *http.Client
	Job             sqlc.Job
	TriggerType     TriggerType
	ManualTriggerID *uuid.UUID
}

type ExecutionTask struct {
	config ExecutionTaskConfig
}

func NewExecutionTask(config ExecutionTaskConfig) *ExecutionTask {
	return &ExecutionTask{config: config}
}

func (t *ExecutionTask) Process(ctx context.Context, processContext *taskworker.ProcessContext) error {
	job := t.config.Job
	now := time.Now().UTC()

	run, err := t.config.Queries.CreateJobRun(ctx, sqlc.CreateJobRunParams{
		ID:             uuid.New(),
		JobID:          job.ID,
		TriggerType:    string(t.config.TriggerType),
		AttemptNumber:  1,
		Status:         "running",
		RequestMethod:  job.Method,
		RequestURL:     job.URL,
		RequestHeaders: job.Headers,
		RequestBody:    job.Body,
		StartedAt:      now,
	})
	if err != nil {
		return fmt.Errorf("create job run: %w", err)
	}

	maxAttempts := int(job.RetryMax) + 1
	var (
		lastResponseStatusCode *int32
		lastResponseHeaders    []byte
		lastResponseBody       *string
		lastError              error
	)

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			if err := t.config.Queries.UpdateRunningAttempt(ctx, run.ID, int32(attempt)); err != nil {
				return fmt.Errorf("update run attempt: %w", err)
			}
		}

		responseStatusCode, responseHeaders, responseBody, err := t.executeHTTPAttempt(ctx)
		lastError = err
		lastResponseStatusCode = responseStatusCode
		lastResponseHeaders = responseHeaders
		lastResponseBody = responseBody

		if err == nil && responseStatusCode != nil && *responseStatusCode >= 200 && *responseStatusCode < 300 {
			finishedAt := time.Now().UTC()
			statusCode := int32(*responseStatusCode)
			_, err := t.config.Queries.CompleteRunSuccess(ctx, sqlc.CompleteRunSuccessParams{
				RunID:              run.ID,
				AttemptNumber:      int32(attempt),
				ResponseStatusCode: statusCode,
				ResponseHeaders:    responseHeaders,
				ResponseBody:       stringValue(responseBody),
				FinishedAt:         finishedAt,
			})
			if err != nil {
				return fmt.Errorf("complete successful run: %w", err)
			}

			if err := t.completeJobExecution(ctx, finishedAt); err != nil {
				return err
			}

			_ = processContext.Logger(fmt.Sprintf("job completed successfully: job_id=%s run_id=%s", job.ID, run.ID))
			return nil
		}

		if attempt < maxAttempts {
			backoff := time.Duration(1<<attempt) * time.Second
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(backoff):
			}
		}
	}

	finishedAt := time.Now().UTC()
	failureMessage := "request failed"
	if lastError != nil {
		failureMessage = lastError.Error()
	}

	_, err = t.config.Queries.CompleteRunFailure(ctx, sqlc.CompleteRunFailureParams{
		RunID:              run.ID,
		AttemptNumber:      int32(maxAttempts),
		ResponseStatusCode: lastResponseStatusCode,
		ResponseHeaders:    lastResponseHeaders,
		ResponseBody:       lastResponseBody,
		ErrorMessage:       failureMessage,
		FinishedAt:         finishedAt,
	})
	if err != nil {
		return fmt.Errorf("complete failed run: %w", err)
	}

	if err := t.completeJobExecution(ctx, finishedAt); err != nil {
		return err
	}

	_ = processContext.Logger(fmt.Sprintf("job run failed: job_id=%s run_id=%s error=%s", job.ID, run.ID, failureMessage))
	return nil
}

func (t *ExecutionTask) executeHTTPAttempt(ctx context.Context) (*int32, []byte, *string, error) {
	job := t.config.Job

	requestContext, cancel := context.WithTimeout(ctx, time.Duration(job.TimeoutSeconds)*time.Second)
	defer cancel()

	var body io.Reader
	if len(job.Body) > 0 {
		body = bytes.NewReader(job.Body)
	}

	httpRequest, err := http.NewRequestWithContext(requestContext, job.Method, job.URL, body)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("build request: %w", err)
	}

	var headers map[string]string
	if len(job.Headers) > 0 {
		if err := json.Unmarshal(job.Headers, &headers); err != nil {
			return nil, nil, nil, fmt.Errorf("decode headers: %w", err)
		}
	}
	for key, value := range headers {
		httpRequest.Header.Set(key, value)
	}

	httpResponse, err := t.config.HTTPClient.Do(httpRequest)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("execute request: %w", err)
	}
	defer httpResponse.Body.Close()

	responseBodyBytes, readErr := io.ReadAll(io.LimitReader(httpResponse.Body, 1<<20))
	if readErr != nil {
		return nil, nil, nil, fmt.Errorf("read response body: %w", readErr)
	}

	responseHeadersBytes, err := json.Marshal(httpResponse.Header)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("encode response headers: %w", err)
	}

	statusCode := int32(httpResponse.StatusCode)
	responseBody := truncateString(string(responseBodyBytes), 65535)

	if statusCode < 200 || statusCode >= 300 {
		failure := fmt.Errorf("received non-2xx status code %d", statusCode)
		return &statusCode, responseHeadersBytes, &responseBody, failure
	}

	return &statusCode, responseHeadersBytes, &responseBody, nil
}

func (t *ExecutionTask) completeJobExecution(ctx context.Context, finishedAt time.Time) error {
	if t.config.TriggerType == TriggerTypeScheduled {
		nextRunAt, err := scheduler.NextRun(t.config.Job.CronExpression, t.config.Job.Timezone, finishedAt)
		if err != nil {
			_ = t.config.Queries.ReleaseClaim(ctx, t.config.Job.ID)
			return fmt.Errorf("compute next run: %w", err)
		}

		if err := t.config.Queries.CompleteScheduledJobExecution(ctx, t.config.Job.ID, finishedAt, nextRunAt); err != nil {
			return fmt.Errorf("complete scheduled execution: %w", err)
		}

		return nil
	}

	if err := t.config.Queries.CompleteManualJobExecution(ctx, t.config.Job.ID, finishedAt); err != nil {
		return fmt.Errorf("complete manual execution: %w", err)
	}

	if t.config.ManualTriggerID != nil {
		if err := t.config.Queries.CompleteManualTrigger(ctx, *t.config.ManualTriggerID); err != nil {
			return fmt.Errorf("complete manual trigger: %w", err)
		}
	}

	return nil
}

func truncateString(value string, limit int) string {
	if len(value) <= limit {
		return value
	}
	return value[:limit]
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
