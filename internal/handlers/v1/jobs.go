package v1

import (
	"encoding/json"
	"net/http"

	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/brian-nunez/go-echo-starter-template/internal/db/sqlc"
	"github.com/brian-nunez/go-echo-starter-template/internal/jobs"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type JobsHandler struct {
	service *jobs.Service
}

func NewJobsHandler(service *jobs.Service) *JobsHandler {
	return &JobsHandler{service: service}
}

type createJobRequest struct {
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

type patchJobRequest struct {
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

func (h *JobsHandler) CreateJob(c echo.Context) error {
	var request createJobRequest
	if err := decodeJSONBody(c, &request); err != nil {
		return err
	}

	created, err := h.service.Create(c.Request().Context(), auth.MustPrincipal(c), jobs.CreateJobInput{
		Name:           request.Name,
		Method:         request.Method,
		URL:            request.URL,
		Headers:        request.Headers,
		Body:           request.Body,
		CronExpression: request.CronExpression,
		Timezone:       request.Timezone,
		RetryMax:       request.RetryMax,
		TimeoutSeconds: request.TimeoutSeconds,
		Enabled:        request.Enabled,
	})
	if err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusCreated, map[string]any{
		"job": toJobResponse(created),
	})
}

func (h *JobsHandler) ListJobs(c echo.Context) error {
	items, err := h.service.List(c.Request().Context(), auth.MustPrincipal(c))
	if err != nil {
		return writeAPIError(c, err)
	}

	response := make([]any, 0, len(items))
	for _, item := range items {
		response = append(response, toJobResponse(item))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"jobs": response,
	})
}

func (h *JobsHandler) GetJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	job, err := h.service.Get(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"job": toJobResponse(job),
	})
}

func (h *JobsHandler) UpdateJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	var request patchJobRequest
	if err := decodeJSONBody(c, &request); err != nil {
		return err
	}

	updated, err := h.service.UpdatePatch(c.Request().Context(), auth.MustPrincipal(c), jobID, jobs.PatchJobInput{
		Name:           request.Name,
		Method:         request.Method,
		URL:            request.URL,
		Headers:        request.Headers,
		Body:           request.Body,
		CronExpression: request.CronExpression,
		Timezone:       request.Timezone,
		RetryMax:       request.RetryMax,
		TimeoutSeconds: request.TimeoutSeconds,
		Enabled:        request.Enabled,
	})
	if err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"job": toJobResponse(updated),
	})
}

func (h *JobsHandler) DeleteJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	if err := h.service.Delete(c.Request().Context(), auth.MustPrincipal(c), jobID); err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"message": "job deleted",
	})
}

func (h *JobsHandler) PauseJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	job, err := h.service.Pause(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"job": toJobResponse(job)})
}

func (h *JobsHandler) ResumeJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	job, err := h.service.Resume(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{"job": toJobResponse(job)})
}

func (h *JobsHandler) TriggerJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	if err := h.service.Trigger(c.Request().Context(), auth.MustPrincipal(c), jobID); err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusAccepted, map[string]any{
		"message": "manual trigger accepted",
	})
}

func toJobResponse(job sqlc.Job) map[string]any {
	headers := map[string]string{}
	if len(job.Headers) > 0 {
		_ = json.Unmarshal(job.Headers, &headers)
	}

	var body any
	if len(job.Body) > 0 {
		_ = json.Unmarshal(job.Body, &body)
	}

	return map[string]any{
		"id":              job.ID,
		"name":            job.Name,
		"method":          job.Method,
		"url":             job.URL,
		"headers":         headers,
		"body":            body,
		"cron_expression": job.CronExpression,
		"timezone":        job.Timezone,
		"retry_max":       job.RetryMax,
		"timeout_seconds": job.TimeoutSeconds,
		"enabled":         job.Enabled,
		"next_run_at":     job.NextRunAt,
		"last_run_at":     job.LastRunAt,
		"created_at":      job.CreatedAt,
		"updated_at":      job.UpdatedAt,
		"tenant_id":       job.TenantID,
		"created_by":      job.CreatedBy,
	}
}
