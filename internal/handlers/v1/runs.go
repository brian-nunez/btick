package v1

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/brian-nunez/btick/internal/auth"
	"github.com/brian-nunez/btick/internal/db/sqlc"
	"github.com/brian-nunez/btick/internal/runs"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type RunsHandler struct {
	service *runs.Service
}

func NewRunsHandler(service *runs.Service) *RunsHandler {
	return &RunsHandler{service: service}
}

func (h *RunsHandler) ListJobRuns(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid jobId")
	}

	limit := int32(100)
	if rawLimit := c.QueryParam("limit"); rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err != nil {
			return echo.NewHTTPError(http.StatusBadRequest, "invalid limit")
		}
		limit = int32(parsed)
	}

	runsList, err := h.service.ListByJob(c.Request().Context(), auth.MustPrincipal(c), jobID, limit)
	if err != nil {
		return writeAPIError(c, err)
	}

	response := make([]any, 0, len(runsList))
	for _, run := range runsList {
		response = append(response, toRunResponse(run))
	}

	return c.JSON(http.StatusOK, map[string]any{
		"runs": response,
	})
}

func (h *RunsHandler) GetRun(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("runId"))
	if err != nil {
		return echo.NewHTTPError(http.StatusBadRequest, "invalid runId")
	}

	run, err := h.service.Get(c.Request().Context(), auth.MustPrincipal(c), runID)
	if err != nil {
		return writeAPIError(c, err)
	}

	return c.JSON(http.StatusOK, map[string]any{
		"run": toRunResponse(run),
	})
}

func toRunResponse(run sqlc.JobRun) map[string]any {
	requestHeaders := map[string]string{}
	if len(run.RequestHeaders) > 0 {
		_ = json.Unmarshal(run.RequestHeaders, &requestHeaders)
	}

	var requestBody any
	if len(run.RequestBody) > 0 {
		_ = json.Unmarshal(run.RequestBody, &requestBody)
	}

	responseHeaders := map[string][]string{}
	if len(run.ResponseHeaders) > 0 {
		_ = json.Unmarshal(run.ResponseHeaders, &responseHeaders)
	}

	return map[string]any{
		"id":                   run.ID,
		"job_id":               run.JobID,
		"trigger_type":         run.TriggerType,
		"attempt_number":       run.AttemptNumber,
		"status":               run.Status,
		"request_method":       run.RequestMethod,
		"request_url":          run.RequestURL,
		"request_headers":      requestHeaders,
		"request_body":         requestBody,
		"response_status_code": run.ResponseStatusCode,
		"response_headers":     responseHeaders,
		"response_body":        run.ResponseBody,
		"error_message":        run.ErrorMessage,
		"started_at":           run.StartedAt,
		"finished_at":          run.FinishedAt,
	}
}
