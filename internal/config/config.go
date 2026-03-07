package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

type APIConfig struct {
	Port             string
	DatabaseURL      string
	StaticAssetsPath string
	UISessionSecret  string
	UISessionTTL     time.Duration
}

type WorkerConfig struct {
	DatabaseURL      string
	WorkerID         string
	PollInterval     time.Duration
	ClaimStaleAfter  time.Duration
	Concurrency      int
	TaskLogPath      string
	TaskStateDBPath  string
	HTTPTimeout      time.Duration
	MaxClaimsPerTick int
}

func LoadAPIConfig() (APIConfig, error) {
	sessionTTLHours, err := intEnv("UI_SESSION_TTL_HOURS", 12)
	if err != nil {
		return APIConfig{}, err
	}

	cfg := APIConfig{
		Port:             envOrDefault("PORT", "8080"),
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		StaticAssetsPath: envOrDefault("STATIC_ASSETS_PATH", "./assets"),
		UISessionSecret:  envOrDefault("UI_SESSION_SECRET", "dev-session-secret-change-me"),
		UISessionTTL:     time.Duration(sessionTTLHours) * time.Hour,
	}

	if cfg.DatabaseURL == "" {
		return APIConfig{}, fmt.Errorf("DATABASE_URL is required")
	}

	return cfg, nil
}

func LoadWorkerConfig() (WorkerConfig, error) {
	concurrency, err := intEnv("WORKER_CONCURRENCY", 4)
	if err != nil {
		return WorkerConfig{}, err
	}
	maxClaims, err := intEnv("WORKER_MAX_CLAIMS_PER_TICK", 20)
	if err != nil {
		return WorkerConfig{}, err
	}
	pollMS, err := intEnv("WORKER_POLL_INTERVAL_MS", 2000)
	if err != nil {
		return WorkerConfig{}, err
	}
	staleMinutes, err := intEnv("WORKER_STALE_CLAIM_MINUTES", 5)
	if err != nil {
		return WorkerConfig{}, err
	}
	httpTimeoutSeconds, err := intEnv("WORKER_HTTP_TIMEOUT_SECONDS", 90)
	if err != nil {
		return WorkerConfig{}, err
	}

	cfg := WorkerConfig{
		DatabaseURL:      strings.TrimSpace(os.Getenv("DATABASE_URL")),
		WorkerID:         envOrDefault("WORKER_ID", "scheduler-worker-1"),
		PollInterval:     time.Duration(pollMS) * time.Millisecond,
		ClaimStaleAfter:  time.Duration(staleMinutes) * time.Minute,
		Concurrency:      concurrency,
		TaskLogPath:      envOrDefault("WORKER_TASK_LOG_PATH", "./logs/tasks"),
		TaskStateDBPath:  envOrDefault("WORKER_TASK_STATE_DB_PATH", "./data/task_orchestration.db"),
		HTTPTimeout:      time.Duration(httpTimeoutSeconds) * time.Second,
		MaxClaimsPerTick: maxClaims,
	}

	if cfg.DatabaseURL == "" {
		return WorkerConfig{}, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.Concurrency <= 0 {
		return WorkerConfig{}, fmt.Errorf("WORKER_CONCURRENCY must be greater than 0")
	}
	if cfg.MaxClaimsPerTick <= 0 {
		return WorkerConfig{}, fmt.Errorf("WORKER_MAX_CLAIMS_PER_TICK must be greater than 0")
	}

	return cfg, nil
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func intEnv(key string, fallback int) (int, error) {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return 0, fmt.Errorf("invalid integer for %s: %w", key, err)
	}
	return parsed, nil
}
