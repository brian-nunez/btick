package httpserver

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/brian-nunez/btick/internal/apikeys"
	"github.com/brian-nunez/btick/internal/authorization"
	"github.com/brian-nunez/btick/internal/config"
	"github.com/brian-nunez/btick/internal/db"
	"github.com/brian-nunez/btick/internal/db/sqlc"
	v1 "github.com/brian-nunez/btick/internal/handlers/v1"
	"github.com/brian-nunez/btick/internal/jobs"
	"github.com/brian-nunez/btick/internal/runs"
	"github.com/brian-nunez/btick/internal/uiauth"
	"github.com/labstack/echo/v4"
)

type Server interface {
	Start(addr string) error
	Shutdown(ctx context.Context) error
}

type appServer struct {
	echo *echo.Echo
	db   *sql.DB
}

func (s *appServer) Start(addr string) error {
	return s.echo.Start(addr)
}

func (s *appServer) Shutdown(ctx context.Context) error {
	echoErr := s.echo.Shutdown(ctx)
	dbErr := s.db.Close()
	if echoErr != nil {
		return echoErr
	}
	if dbErr != nil {
		return dbErr
	}
	return nil
}

type BootstrapConfig struct {
	AppConfig config.APIConfig
}

func Bootstrap(config BootstrapConfig) (Server, error) {
	database, err := db.OpenPostgres(config.AppConfig.DatabaseURL)
	if err != nil {
		return nil, err
	}

	if err := db.RunMigrations(context.Background(), database, "./migrations"); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("run migrations: %w", err)
	}

	queries := sqlc.New(database)
	authorizer, err := authorization.NewAuthorizer()
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("build authorizer: %w", err)
	}

	jobsService := jobs.NewService(queries, authorizer)
	runsService := runs.NewService(queries, authorizer)
	apiKeysService := apikeys.NewService(queries, authorizer)
	uiAuthService, err := uiauth.NewService(queries, uiauth.Config{
		SessionSecret: config.AppConfig.UISessionSecret,
		SessionTTL:    config.AppConfig.UISessionTTL,
	})
	if err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("build ui auth service: %w", err)
	}

	echoServer := New().
		WithStaticAssets(map[string]string{
			"/assets": config.AppConfig.StaticAssetsPath,
		}).
		WithDefaultMiddleware().
		WithErrorHandler().
		WithRoutes(func(e *echo.Echo) {
			v1.RegisterRoutes(e, v1.Dependencies{
				Database:       database,
				JobsService:    jobsService,
				RunsService:    runsService,
				APIKeysService: apiKeysService,
				UIAuthService:  uiAuthService,
			})
		}).
		WithNotFound().
		Build()

	return &appServer{
		echo: echoServer,
		db:   database,
	}, nil
}
