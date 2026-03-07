package v1

import (
	"database/sql"

	"github.com/brian-nunez/go-echo-starter-template/internal/apikeys"
	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	uihandlers "github.com/brian-nunez/go-echo-starter-template/internal/handlers/v1/ui"
	"github.com/brian-nunez/go-echo-starter-template/internal/jobs"
	"github.com/brian-nunez/go-echo-starter-template/internal/runs"
	"github.com/labstack/echo/v4"
)

type Dependencies struct {
	Database       *sql.DB
	JobsService    *jobs.Service
	RunsService    *runs.Service
	APIKeysService *apikeys.Service
}

func RegisterRoutes(e *echo.Echo, dependencies Dependencies) {
	healthHandler := NewHealthHandler(dependencies.Database)
	uiHandler := uihandlers.NewHandler(dependencies.JobsService, dependencies.RunsService, dependencies.APIKeysService)
	jobsHandler := NewJobsHandler(dependencies.JobsService)
	runsHandler := NewRunsHandler(dependencies.RunsService)
	keysHandler := NewAPIKeysHandler(dependencies.APIKeysService)

	e.GET("/", uiHandler.JobsPage)
	e.GET("/ui/jobs", uiHandler.JobsPage)
	e.GET("/ui/jobs/new", uiHandler.CreateJobPage)
	e.POST("/ui/jobs", uiHandler.CreateJob)
	e.GET("/ui/jobs/:jobId", uiHandler.JobDetailPage)
	e.GET("/ui/jobs/:jobId/edit", uiHandler.EditJobPage)
	e.POST("/ui/jobs/:jobId", uiHandler.UpdateJob)
	e.POST("/ui/jobs/:jobId/pause", uiHandler.PauseJob)
	e.POST("/ui/jobs/:jobId/resume", uiHandler.ResumeJob)
	e.POST("/ui/jobs/:jobId/trigger", uiHandler.TriggerJob)
	e.GET("/ui/jobs/:jobId/runs", uiHandler.JobRunsPage)
	e.GET("/ui/runs/:runId", uiHandler.RunDetailPage)
	e.GET("/ui/api-keys", uiHandler.APIKeysPage)
	e.GET("/ui/api-keys/new", uiHandler.CreateAPIKeyPage)
	e.POST("/ui/api-keys", uiHandler.CreateAPIKey)
	e.POST("/ui/api-keys/:keyId/revoke", uiHandler.RevokeAPIKey)
	e.GET("/ui/partials/jobs-table", uiHandler.JobsTablePartial)

	e.GET("/healthz", healthHandler.Healthz)
	e.GET("/readyz", healthHandler.Readyz)

	v1Group := e.Group("/api/v1")
	v1Group.GET("/health", healthHandler.V1Health)

	protected := v1Group.Group("", auth.APIKeyAuthMiddleware(dependencies.APIKeysService))

	protected.POST("/jobs", jobsHandler.CreateJob)
	protected.GET("/jobs", jobsHandler.ListJobs)
	protected.GET("/jobs/:jobId", jobsHandler.GetJob)
	protected.PATCH("/jobs/:jobId", jobsHandler.UpdateJob)
	protected.DELETE("/jobs/:jobId", jobsHandler.DeleteJob)
	protected.POST("/jobs/:jobId/pause", jobsHandler.PauseJob)
	protected.POST("/jobs/:jobId/resume", jobsHandler.ResumeJob)
	protected.POST("/jobs/:jobId/trigger", jobsHandler.TriggerJob)

	protected.GET("/jobs/:jobId/runs", runsHandler.ListJobRuns)
	protected.GET("/runs/:runId", runsHandler.GetRun)

	protected.POST("/api-keys", keysHandler.CreateAPIKey)
	protected.GET("/api-keys", keysHandler.ListAPIKeys)
	protected.POST("/api-keys/:keyId/revoke", keysHandler.RevokeAPIKey)
}
