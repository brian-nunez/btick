package v1

import (
	"database/sql"

	"github.com/brian-nunez/go-echo-starter-template/internal/apikeys"
	"github.com/brian-nunez/go-echo-starter-template/internal/auth"
	uihandlers "github.com/brian-nunez/go-echo-starter-template/internal/handlers/v1/ui"
	"github.com/brian-nunez/go-echo-starter-template/internal/jobs"
	"github.com/brian-nunez/go-echo-starter-template/internal/runs"
	"github.com/brian-nunez/go-echo-starter-template/internal/uiauth"
	"github.com/labstack/echo/v4"
)

type Dependencies struct {
	Database       *sql.DB
	JobsService    *jobs.Service
	RunsService    *runs.Service
	APIKeysService *apikeys.Service
	UIAuthService  *uiauth.Service
}

func RegisterRoutes(e *echo.Echo, dependencies Dependencies) {
	healthHandler := NewHealthHandler(dependencies.Database)
	uiHandler := uihandlers.NewHandler(dependencies.JobsService, dependencies.RunsService, dependencies.APIKeysService, dependencies.UIAuthService)
	jobsHandler := NewJobsHandler(dependencies.JobsService)
	runsHandler := NewRunsHandler(dependencies.RunsService)
	keysHandler := NewAPIKeysHandler(dependencies.APIKeysService)

	e.Use(uiauth.SessionMiddleware(dependencies.UIAuthService))

	e.GET("/", uiHandler.Root)
	e.GET("/register", uiHandler.RegisterPage, uiauth.RequireGuest)
	e.POST("/register", uiHandler.Register, uiauth.RequireGuest)
	e.GET("/login", uiHandler.LoginPage, uiauth.RequireGuest)
	e.POST("/login", uiHandler.Login, uiauth.RequireGuest)
	e.POST("/logout", uiHandler.Logout, uiauth.RequireLogin(dependencies.UIAuthService))

	protectedUI := e.Group("", uiauth.RequireLogin(dependencies.UIAuthService))
	protectedUI.GET("/ui/jobs", uiHandler.JobsPage)
	protectedUI.GET("/ui/jobs/new", uiHandler.CreateJobPage)
	protectedUI.POST("/ui/jobs", uiHandler.CreateJob)
	protectedUI.GET("/ui/jobs/:jobId", uiHandler.JobDetailPage)
	protectedUI.GET("/ui/jobs/:jobId/edit", uiHandler.EditJobPage)
	protectedUI.POST("/ui/jobs/:jobId", uiHandler.UpdateJob)
	protectedUI.POST("/ui/jobs/:jobId/pause", uiHandler.PauseJob)
	protectedUI.POST("/ui/jobs/:jobId/resume", uiHandler.ResumeJob)
	protectedUI.POST("/ui/jobs/:jobId/trigger", uiHandler.TriggerJob)
	protectedUI.GET("/ui/jobs/:jobId/runs", uiHandler.JobRunsPage)
	protectedUI.GET("/ui/runs/:runId", uiHandler.RunDetailPage)
	protectedUI.GET("/ui/api-keys", uiHandler.APIKeysPage)
	protectedUI.GET("/ui/api-keys/new", uiHandler.CreateAPIKeyPage)
	protectedUI.POST("/ui/api-keys", uiHandler.CreateAPIKey)
	protectedUI.POST("/ui/api-keys/:keyId/revoke", uiHandler.RevokeAPIKey)
	protectedUI.GET("/ui/partials/jobs-table", uiHandler.JobsTablePartial)

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
