package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/brian-nunez/btick/internal/config"
	"github.com/brian-nunez/btick/internal/httpserver"
)

func main() {
	appConfig, err := config.LoadAPIConfig()
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	server, err := httpserver.Bootstrap(httpserver.BootstrapConfig{AppConfig: appConfig})
	if err != nil {
		log.Fatalf("bootstrap server: %v", err)
	}

	address := fmt.Sprintf(":%s", appConfig.Port)
	go func() {
		if err := server.Start(address); err != nil {
			log.Printf("server stopped: %v", err)
		}
	}()

	shutdownSignals := make(chan os.Signal, 1)
	signal.Notify(shutdownSignals, os.Interrupt, syscall.SIGTERM)
	<-shutdownSignals

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("shutdown failed: %v", err)
	}
}
