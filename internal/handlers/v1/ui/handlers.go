package uihandlers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/a-h/templ"
	"github.com/brian-nunez/go-echo-starter-template/internal/apikeys"
	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	"github.com/brian-nunez/go-echo-starter-template/internal/db/sqlc"
	"github.com/brian-nunez/go-echo-starter-template/internal/jobs"
	"github.com/brian-nunez/go-echo-starter-template/internal/runs"
	"github.com/brian-nunez/go-echo-starter-template/internal/uiauth"
	"github.com/brian-nunez/go-echo-starter-template/views/pages/scheduler"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
)

type Handler struct {
	jobsService    *jobs.Service
	runsService    *runs.Service
	apiKeysService *apikeys.Service
	uiAuthService  *uiauth.Service
}

func NewHandler(jobsService *jobs.Service, runsService *runs.Service, apiKeysService *apikeys.Service, uiAuthService *uiauth.Service) *Handler {
	return &Handler{
		jobsService:    jobsService,
		runsService:    runsService,
		apiKeysService: apiKeysService,
		uiAuthService:  uiAuthService,
	}
}

func (h *Handler) Root(c echo.Context) error {
	return render(c, scheduler.LandingPage(scheduler.LandingPageData{
		Authenticated: auth.MustPrincipal(c).SubjectID != "",
	}))
}

func (h *Handler) RegisterPage(c echo.Context) error {
	return render(c, scheduler.AuthPage(scheduler.AuthPageData{
		Title:            "Create your account",
		Subtitle:         "Set up your scheduler admin account.",
		Action:           "/register",
		SubmitLabel:      "Create account",
		SecondaryText:    "Already have an account?",
		SecondaryURL:     "/login",
		SecondaryLabel:   "Sign in",
		BackURL:          "/",
		BackLabel:        "Back to home",
		EmailPlaceholder: "you@company.com",
	}))
}

func (h *Handler) LoginPage(c echo.Context) error {
	return render(c, scheduler.AuthPage(scheduler.AuthPageData{
		Title:            "Welcome back",
		Subtitle:         "Sign in to manage scheduled jobs and run history.",
		Action:           "/login",
		SubmitLabel:      "Sign in",
		SecondaryText:    "Need an account?",
		SecondaryURL:     "/register",
		SecondaryLabel:   "Create account",
		BackURL:          "/",
		BackLabel:        "Back to home",
		EmailPlaceholder: "you@company.com",
	}))
}

func (h *Handler) Register(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	principal, err := h.uiAuthService.Register(c.Request().Context(), email, password)
	if err != nil {
		return render(c, scheduler.AuthPage(scheduler.AuthPageData{
			Title:            "Create your account",
			Subtitle:         "Set up your scheduler admin account.",
			Action:           "/register",
			SubmitLabel:      "Create account",
			SecondaryText:    "Already have an account?",
			SecondaryURL:     "/login",
			SecondaryLabel:   "Sign in",
			BackURL:          "/",
			BackLabel:        "Back to home",
			EmailPlaceholder: "you@company.com",
			Email:            email,
			Error:            err.Error(),
		}))
	}

	cookie, err := h.uiAuthService.NewSessionCookie(principal, c.Scheme() == "https")
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to create session")
	}
	c.SetCookie(cookie)
	return c.Redirect(http.StatusSeeOther, "/ui/jobs")
}

func (h *Handler) Login(c echo.Context) error {
	email := strings.TrimSpace(c.FormValue("email"))
	password := c.FormValue("password")

	principal, err := h.uiAuthService.Login(c.Request().Context(), email, password)
	if err != nil {
		return render(c, scheduler.AuthPage(scheduler.AuthPageData{
			Title:            "Welcome back",
			Subtitle:         "Sign in to manage scheduled jobs and run history.",
			Action:           "/login",
			SubmitLabel:      "Sign in",
			SecondaryText:    "Need an account?",
			SecondaryURL:     "/register",
			SecondaryLabel:   "Create account",
			BackURL:          "/",
			BackLabel:        "Back to home",
			EmailPlaceholder: "you@company.com",
			Email:            email,
			Error:            err.Error(),
		}))
	}

	cookie, err := h.uiAuthService.NewSessionCookie(principal, c.Scheme() == "https")
	if err != nil {
		return c.String(http.StatusInternalServerError, "failed to create session")
	}
	c.SetCookie(cookie)
	return c.Redirect(http.StatusSeeOther, "/ui/jobs")
}

func (h *Handler) Logout(c echo.Context) error {
	c.SetCookie(h.uiAuthService.ClearSessionCookie(c.Scheme() == "https"))
	return c.Redirect(http.StatusSeeOther, "/login")
}

func (h *Handler) JobsPage(c echo.Context) error {
	principal := auth.MustPrincipal(c)
	jobList, err := h.jobsService.List(c.Request().Context(), principal)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, scheduler.JobsPage(scheduler.JobsPageData{
		Title: "Scheduled Jobs",
		Jobs:  toUIJobs(jobList),
	}))
}

func (h *Handler) JobsTablePartial(c echo.Context) error {
	principal := auth.MustPrincipal(c)
	jobList, err := h.jobsService.List(c.Request().Context(), principal)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return render(c, scheduler.JobsTable(scheduler.JobsTableData{Jobs: toUIJobs(jobList)}))
}

func (h *Handler) CreateJobPage(c echo.Context) error {
	return render(c, scheduler.JobFormPage(scheduler.JobFormPageData{
		Title:       "Create Job",
		SubmitLabel: "Create Job",
		FormAction:  "/ui/jobs",
		Job:         defaultUIJobForm(),
	}))
}

func (h *Handler) CreateJob(c echo.Context) error {
	input, uiForm, validationMessage := parseJobForm(c)
	if validationMessage != "" {
		return render(c, scheduler.JobFormPage(scheduler.JobFormPageData{
			Title:       "Create Job",
			SubmitLabel: "Create Job",
			FormAction:  "/ui/jobs",
			Job:         uiForm,
			Error:       validationMessage,
		}))
	}

	created, err := h.jobsService.Create(c.Request().Context(), auth.MustPrincipal(c), input)
	if err != nil {
		return render(c, scheduler.JobFormPage(scheduler.JobFormPageData{
			Title:       "Create Job",
			SubmitLabel: "Create Job",
			FormAction:  "/ui/jobs",
			Job:         uiForm,
			Error:       err.Error(),
		}))
	}

	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/jobs/%s", created.ID))
}

func (h *Handler) EditJobPage(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid job id")
	}

	job, err := h.jobsService.Get(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return c.String(http.StatusNotFound, "job not found")
	}

	return render(c, scheduler.JobFormPage(scheduler.JobFormPageData{
		Title:       "Edit Job",
		SubmitLabel: "Save Changes",
		FormAction:  fmt.Sprintf("/ui/jobs/%s", job.ID),
		Job:         toUIJobForm(job),
	}))
}

func (h *Handler) UpdateJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid job id")
	}

	input, uiForm, validationMessage := parseJobForm(c)
	if validationMessage != "" {
		return render(c, scheduler.JobFormPage(scheduler.JobFormPageData{
			Title:       "Edit Job",
			SubmitLabel: "Save Changes",
			FormAction:  fmt.Sprintf("/ui/jobs/%s", jobID),
			Job:         uiForm,
			Error:       validationMessage,
		}))
	}

	updated, err := h.jobsService.UpdatePatch(c.Request().Context(), auth.MustPrincipal(c), jobID, jobs.PatchJobInput{
		Name:           &input.Name,
		Method:         &input.Method,
		URL:            &input.URL,
		Headers:        &input.Headers,
		Body:           &input.Body,
		CronExpression: &input.CronExpression,
		Timezone:       &input.Timezone,
		RetryMax:       &input.RetryMax,
		TimeoutSeconds: &input.TimeoutSeconds,
		Enabled:        &input.Enabled,
	})
	if err != nil {
		return render(c, scheduler.JobFormPage(scheduler.JobFormPageData{
			Title:       "Edit Job",
			SubmitLabel: "Save Changes",
			FormAction:  fmt.Sprintf("/ui/jobs/%s", jobID),
			Job:         uiForm,
			Error:       err.Error(),
		}))
	}

	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/jobs/%s", updated.ID))
}

func (h *Handler) JobDetailPage(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid job id")
	}

	job, err := h.jobsService.Get(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return c.String(http.StatusNotFound, "job not found")
	}

	runsList, _ := h.runsService.ListByJob(c.Request().Context(), auth.MustPrincipal(c), job.ID, 20)

	return render(c, scheduler.JobDetailPage(scheduler.JobDetailPageData{
		Job:  toUIJob(job),
		Runs: toUIRuns(runsList),
	}))
}

func (h *Handler) JobRunsPage(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid job id")
	}

	job, err := h.jobsService.Get(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return c.String(http.StatusNotFound, "job not found")
	}

	runsList, err := h.runsService.ListByJob(c.Request().Context(), auth.MustPrincipal(c), jobID, 200)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, scheduler.JobRunsPage(scheduler.JobRunsPageData{
		Job:  toUIJob(job),
		Runs: toUIRuns(runsList),
	}))
}

func (h *Handler) RunDetailPage(c echo.Context) error {
	runID, err := uuid.Parse(c.Param("runId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid run id")
	}

	run, err := h.runsService.Get(c.Request().Context(), auth.MustPrincipal(c), runID)
	if err != nil {
		return c.String(http.StatusNotFound, "run not found")
	}

	return render(c, scheduler.RunDetailPage(scheduler.RunDetailPageData{Run: toUIRun(run)}))
}

func (h *Handler) PauseJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid job id")
	}
	_, err = h.jobsService.Pause(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if isHTMX(c) {
		return h.JobsTablePartial(c)
	}
	return c.Redirect(http.StatusSeeOther, "/ui/jobs")
}

func (h *Handler) ResumeJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid job id")
	}
	_, err = h.jobsService.Resume(c.Request().Context(), auth.MustPrincipal(c), jobID)
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if isHTMX(c) {
		return h.JobsTablePartial(c)
	}
	return c.Redirect(http.StatusSeeOther, "/ui/jobs")
}

func (h *Handler) TriggerJob(c echo.Context) error {
	jobID, err := uuid.Parse(c.Param("jobId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid job id")
	}
	if err := h.jobsService.Trigger(c.Request().Context(), auth.MustPrincipal(c), jobID); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	if isHTMX(c) {
		return h.JobsTablePartial(c)
	}
	return c.Redirect(http.StatusSeeOther, fmt.Sprintf("/ui/jobs/%s", jobID))
}

func (h *Handler) APIKeysPage(c echo.Context) error {
	keys, err := h.apiKeysService.List(c.Request().Context(), auth.MustPrincipal(c))
	if err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}

	return render(c, scheduler.APIKeysPage(scheduler.APIKeysPageData{
		Keys: toUIKeys(keys),
	}))
}

func (h *Handler) CreateAPIKeyPage(c echo.Context) error {
	return render(c, scheduler.CreateAPIKeyPage(scheduler.CreateAPIKeyPageData{}))
}

func (h *Handler) CreateAPIKey(c echo.Context) error {
	name := strings.TrimSpace(c.FormValue("name"))
	scopes := normalizeScopes(c.FormValue("scopes"))

	var expiresAt *time.Time
	if rawExpiresAt := strings.TrimSpace(c.FormValue("expires_at")); rawExpiresAt != "" {
		parsed, err := time.Parse(time.RFC3339, rawExpiresAt)
		if err != nil {
			return render(c, scheduler.CreateAPIKeyPage(scheduler.CreateAPIKeyPageData{
				Error:  "Expires At must be RFC3339 (example: 2026-12-31T23:59:59Z)",
				Name:   name,
				Scopes: strings.Join(scopes, ","),
			}))
		}
		expiresAt = &parsed
	}

	result, err := h.apiKeysService.Create(c.Request().Context(), auth.MustPrincipal(c), apikeys.CreateAPIKeyInput{
		Name:      name,
		Scopes:    scopes,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return render(c, scheduler.CreateAPIKeyPage(scheduler.CreateAPIKeyPageData{
			Error:  err.Error(),
			Name:   name,
			Scopes: strings.Join(scopes, ","),
		}))
	}

	return render(c, scheduler.CreateAPIKeyPage(scheduler.CreateAPIKeyPageData{
		RawKey: result.RawKey,
	}))
}

func (h *Handler) RevokeAPIKey(c echo.Context) error {
	keyID, err := uuid.Parse(c.Param("keyId"))
	if err != nil {
		return c.String(http.StatusBadRequest, "invalid key id")
	}

	if _, err := h.apiKeysService.Revoke(c.Request().Context(), auth.MustPrincipal(c), keyID); err != nil {
		return c.String(http.StatusInternalServerError, err.Error())
	}
	return c.Redirect(http.StatusSeeOther, "/ui/api-keys")
}

func parseJobForm(c echo.Context) (jobs.CreateJobInput, scheduler.UIJobForm, string) {
	name := strings.TrimSpace(c.FormValue("name"))
	method := strings.ToUpper(strings.TrimSpace(c.FormValue("method")))
	urlValue := strings.TrimSpace(c.FormValue("url"))
	cronExpression := strings.TrimSpace(c.FormValue("cron_expression"))
	timezone := strings.TrimSpace(c.FormValue("timezone"))
	if timezone == "" {
		timezone = "UTC"
	}
	retryMax, _ := strconv.Atoi(strings.TrimSpace(c.FormValue("retry_max")))
	timeoutSeconds, _ := strconv.Atoi(strings.TrimSpace(c.FormValue("timeout_seconds")))
	enabled := c.FormValue("enabled") == "on"

	headersJSON := strings.TrimSpace(c.FormValue("headers_json"))
	if headersJSON == "" {
		headersJSON = "{}"
	}
	bodyJSON := strings.TrimSpace(c.FormValue("body_json"))
	if bodyJSON == "" {
		bodyJSON = "null"
	}

	headers := map[string]string{}
	if err := json.Unmarshal([]byte(headersJSON), &headers); err != nil {
		return jobs.CreateJobInput{}, scheduler.UIJobForm{
			Name:           name,
			Method:         method,
			URL:            urlValue,
			HeadersJSON:    headersJSON,
			BodyJSON:       bodyJSON,
			CronExpression: cronExpression,
			Timezone:       timezone,
			RetryMax:       int32(retryMax),
			TimeoutSeconds: int32(timeoutSeconds),
			Enabled:        enabled,
		}, "Headers must be valid JSON object"
	}

	body := json.RawMessage(bodyJSON)
	if bodyJSON != "null" {
		var raw any
		if err := json.Unmarshal(body, &raw); err != nil {
			return jobs.CreateJobInput{}, scheduler.UIJobForm{
				Name:           name,
				Method:         method,
				URL:            urlValue,
				HeadersJSON:    headersJSON,
				BodyJSON:       bodyJSON,
				CronExpression: cronExpression,
				Timezone:       timezone,
				RetryMax:       int32(retryMax),
				TimeoutSeconds: int32(timeoutSeconds),
				Enabled:        enabled,
			}, "Body must be valid JSON"
		}
	}

	input := jobs.CreateJobInput{
		Name:           name,
		Method:         method,
		URL:            urlValue,
		Headers:        headers,
		Body:           body,
		CronExpression: cronExpression,
		Timezone:       timezone,
		RetryMax:       int32(retryMax),
		TimeoutSeconds: int32(timeoutSeconds),
		Enabled:        enabled,
	}

	form := scheduler.UIJobForm{
		Name:           name,
		Method:         method,
		URL:            urlValue,
		HeadersJSON:    headersJSON,
		BodyJSON:       bodyJSON,
		CronExpression: cronExpression,
		Timezone:       timezone,
		RetryMax:       int32(retryMax),
		TimeoutSeconds: int32(timeoutSeconds),
		Enabled:        enabled,
	}

	return input, form, ""
}

func toUIJob(item sqlc.Job) scheduler.UIJob {
	headersJSON := "{}"
	if len(item.Headers) > 0 {
		headersJSON = string(item.Headers)
	}
	bodyJSON := "null"
	if len(item.Body) > 0 {
		bodyJSON = string(item.Body)
	}
	return scheduler.UIJob{
		ID:             item.ID.String(),
		Name:           item.Name,
		Method:         item.Method,
		URL:            item.URL,
		HeadersJSON:    headersJSON,
		BodyJSON:       bodyJSON,
		CronExpression: item.CronExpression,
		Timezone:       item.Timezone,
		RetryMax:       item.RetryMax,
		TimeoutSeconds: item.TimeoutSeconds,
		Enabled:        item.Enabled,
		NextRunAt:      item.NextRunAt,
		LastRunAt:      item.LastRunAt,
		CreatedAt:      item.CreatedAt,
		UpdatedAt:      item.UpdatedAt,
	}
}

func toUIJobs(items []sqlc.Job) []scheduler.UIJob {
	out := make([]scheduler.UIJob, 0, len(items))
	for _, item := range items {
		out = append(out, toUIJob(item))
	}
	return out
}

func toUIRun(item sqlc.JobRun) scheduler.UIRun {
	responseStatus := "-"
	if item.ResponseStatusCode != nil {
		responseStatus = strconv.Itoa(int(*item.ResponseStatusCode))
	}
	responseBody := ""
	if item.ResponseBody != nil {
		responseBody = *item.ResponseBody
	}
	errorMessage := ""
	if item.ErrorMessage != nil {
		errorMessage = *item.ErrorMessage
	}
	return scheduler.UIRun{
		ID:              item.ID.String(),
		JobID:           item.JobID.String(),
		TriggerType:     item.TriggerType,
		AttemptNumber:   item.AttemptNumber,
		Status:          item.Status,
		RequestMethod:   item.RequestMethod,
		RequestURL:      item.RequestURL,
		RequestHeaders:  prettyJSON(item.RequestHeaders),
		RequestBody:     prettyJSON(item.RequestBody),
		ResponseStatus:  responseStatus,
		ResponseHeaders: prettyJSON(item.ResponseHeaders),
		ResponseBody:    responseBody,
		ErrorMessage:    errorMessage,
		StartedAt:       item.StartedAt,
		FinishedAt:      item.FinishedAt,
	}
}

func toUIRuns(items []sqlc.JobRun) []scheduler.UIRun {
	out := make([]scheduler.UIRun, 0, len(items))
	for _, item := range items {
		out = append(out, toUIRun(item))
	}
	return out
}

func toUIKeys(items []sqlc.APIKey) []scheduler.UIAPIKey {
	out := make([]scheduler.UIAPIKey, 0, len(items))
	for _, item := range items {
		var scopes []string
		if len(item.Scopes) > 0 {
			_ = json.Unmarshal(item.Scopes, &scopes)
		}
		out = append(out, scheduler.UIAPIKey{
			ID:         item.ID.String(),
			Name:       item.Name,
			KeyPrefix:  item.KeyPrefix,
			Scopes:     strings.Join(scopes, ", "),
			CreatedAt:  item.CreatedAt,
			RevokedAt:  item.RevokedAt,
			ExpiresAt:  item.ExpiresAt,
			LastUsedAt: item.LastUsedAt,
		})
	}
	return out
}

func defaultUIJobForm() scheduler.UIJobForm {
	return scheduler.UIJobForm{
		Method:         http.MethodPost,
		HeadersJSON:    "{}",
		BodyJSON:       "null",
		CronExpression: "0 * * * *",
		Timezone:       "UTC",
		RetryMax:       0,
		TimeoutSeconds: 60,
		Enabled:        true,
	}
}

func toUIJobForm(job sqlc.Job) scheduler.UIJobForm {
	uiJob := toUIJob(job)
	return scheduler.UIJobForm{
		Name:           uiJob.Name,
		Method:         uiJob.Method,
		URL:            uiJob.URL,
		HeadersJSON:    uiJob.HeadersJSON,
		BodyJSON:       uiJob.BodyJSON,
		CronExpression: uiJob.CronExpression,
		Timezone:       uiJob.Timezone,
		RetryMax:       uiJob.RetryMax,
		TimeoutSeconds: uiJob.TimeoutSeconds,
		Enabled:        uiJob.Enabled,
	}
}

func prettyJSON(value []byte) string {
	if len(value) == 0 {
		return ""
	}
	var pretty bytes.Buffer
	if err := json.Indent(&pretty, value, "", "  "); err != nil {
		return string(value)
	}
	return pretty.String()
}

func isHTMX(c echo.Context) bool {
	return strings.EqualFold(c.Request().Header.Get("HX-Request"), "true")
}

func normalizeScopes(raw string) []string {
	parts := strings.Split(raw, ",")
	scopes := make([]string, 0, len(parts))
	for _, part := range parts {
		scope := strings.TrimSpace(part)
		if scope == "" {
			continue
		}
		scopes = append(scopes, scope)
	}
	return scopes
}

func render(c echo.Context, component templ.Component) error {
	c.Response().Header().Set(echo.HeaderContentType, echo.MIMETextHTMLCharsetUTF8)
	return component.Render(context.Background(), c.Response().Writer)
}
