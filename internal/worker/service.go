package worker

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/brian-nunez/btick/internal/config"
	"github.com/brian-nunez/btick/internal/db/sqlc"
	taskworker "github.com/brian-nunez/task-orchestration"
)

type Service struct {
	config     config.WorkerConfig
	queries    *sqlc.Queries
	logger     *log.Logger
	httpClient *http.Client
	pool       *taskworker.WorkerPool
}

func NewService(config config.WorkerConfig, queries *sqlc.Queries, logger *log.Logger) *Service {
	pool := taskworker.NewWorkerPool(taskworker.WorkerPoolConfig{
		Concurrency:  config.Concurrency,
		LogPath:      config.TaskLogPath,
		DatabasePath: ":memory:",
	})

	return &Service{
		config:  config,
		queries: queries,
		logger:  logger,
		httpClient: &http.Client{
			Timeout: config.HTTPTimeout,
		},
		pool: pool,
	}
}

func (s *Service) Run(ctx context.Context) error {
	if err := ensureTaskStatePath(s.config.TaskStateDBPath); err != nil {
		return fmt.Errorf("prepare task state database path: %w", err)
	}

	if err := s.pool.Start(); err != nil {
		return fmt.Errorf("start task worker pool: %w", err)
	}
	defer s.pool.Stop()

	ticker := time.NewTicker(s.config.PollInterval)
	defer ticker.Stop()

	s.logger.Printf("worker started: worker_id=%s poll_interval=%s concurrency=%d", s.config.WorkerID, s.config.PollInterval, s.config.Concurrency)

	for {
		select {
		case <-ctx.Done():
			s.logger.Println("worker received shutdown signal")
			return nil
		case <-ticker.C:
			s.poll(ctx)
		}
	}
}

func (s *Service) poll(ctx context.Context) {
	staleInterval := toPostgresInterval(s.config.ClaimStaleAfter)

	s.claimManualTriggers(ctx, staleInterval)
	s.claimScheduledJobs(ctx, staleInterval)
}

func (s *Service) claimManualTriggers(ctx context.Context, staleInterval string) {
	for i := 0; i < s.config.MaxClaimsPerTick; i++ {
		trigger, err := s.queries.ClaimManualTrigger(ctx, s.config.WorkerID, staleInterval)
		if err != nil {
			s.logger.Printf("manual trigger claim error: %v", err)
			return
		}
		if trigger == nil {
			return
		}

		job, err := s.queries.GetManualTriggerJob(ctx, trigger.ID)
		if err != nil {
			s.logger.Printf("load manual trigger job error: trigger_id=%s err=%v", trigger.ID, err)
			_ = s.queries.ResetManualTriggerClaim(ctx, trigger.ID)
			continue
		}

		task := NewExecutionTask(ExecutionTaskConfig{
			Queries:         s.queries,
			Logger:          s.logger,
			HTTPClient:      s.httpClient,
			Job:             job,
			TriggerType:     TriggerTypeManual,
			ManualTriggerID: &trigger.ID,
		})

		if _, err := s.pool.AddTask(task); err != nil {
			s.logger.Printf("enqueue manual task error: trigger_id=%s job_id=%s err=%v", trigger.ID, job.ID, err)
			_ = s.queries.ResetManualTriggerClaim(ctx, trigger.ID)
		}
	}
}

func (s *Service) claimScheduledJobs(ctx context.Context, staleInterval string) {
	for i := 0; i < s.config.MaxClaimsPerTick; i++ {
		job, err := s.queries.ClaimDueJob(ctx, s.config.WorkerID, staleInterval)
		if err != nil {
			s.logger.Printf("scheduled claim error: %v", err)
			return
		}
		if job == nil {
			return
		}

		task := NewExecutionTask(ExecutionTaskConfig{
			Queries:     s.queries,
			Logger:      s.logger,
			HTTPClient:  s.httpClient,
			Job:         *job,
			TriggerType: TriggerTypeScheduled,
		})

		if _, err := s.pool.AddTask(task); err != nil {
			s.logger.Printf("enqueue scheduled task error: job_id=%s err=%v", job.ID, err)
			_ = s.queries.ReleaseClaim(ctx, job.ID)
		}
	}
}

func toPostgresInterval(duration time.Duration) string {
	if duration <= 0 {
		return "5 minutes"
	}
	if duration%time.Minute == 0 {
		return fmt.Sprintf("%d minutes", int(duration/time.Minute))
	}
	return fmt.Sprintf("%d seconds", int(duration/time.Second))
}

func ensureTaskStatePath(path string) error {
	trimmedPath := strings.TrimSpace(path)
	if trimmedPath == "" || trimmedPath == ":memory:" {
		return nil
	}

	if strings.HasPrefix(trimmedPath, "file:") {
		trimmedPath = strings.TrimPrefix(trimmedPath, "file:")
		if index := strings.Index(trimmedPath, "?"); index >= 0 {
			trimmedPath = trimmedPath[:index]
		}
	}

	if trimmedPath == "" {
		return nil
	}

	dir := filepath.Dir(trimmedPath)
	if dir == "." || dir == "" {
		return nil
	}

	return os.MkdirAll(dir, 0o755)
}
