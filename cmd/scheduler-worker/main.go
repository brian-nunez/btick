package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brian-nunez/btick/internal/config"
	"github.com/brian-nunez/btick/internal/db"
	"github.com/brian-nunez/btick/internal/db/sqlc"
	"github.com/brian-nunez/btick/internal/worker"
)

func main() {
	cfg, err := config.LoadWorkerConfig()
	if err != nil {
		log.Fatalf("load worker config: %v", err)
	}

	database, err := db.OpenPostgres(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open database: %v", err)
	}
	defer database.Close()

	if err := db.RunMigrations(context.Background(), database, "./migrations"); err != nil {
		log.Fatalf("run migrations: %v", err)
	}

	queries := sqlc.New(database)
	logger := log.New(os.Stdout, "[scheduler-worker] ", log.LstdFlags|log.LUTC)
	service := worker.NewService(cfg, queries, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-shutdownSignals
		cancel()
	}()

	if err := service.Run(ctx); err != nil {
		log.Fatalf("worker run error: %v", err)
	}

	// Allow in-flight logs flush from task worker.
	time.Sleep(200 * time.Millisecond)
}
