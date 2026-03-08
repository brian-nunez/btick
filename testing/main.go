package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/brian-nunez/btick/sdk/go/scheduler"
)

func main() {
	baseURL := envOrDefault("SCHEDULER_BASE_URL", "http://localhost:8080")
	apiKey := strings.TrimSpace(os.Getenv("SCHEDULER_API_KEY"))
	if apiKey == "" {
		log.Fatal("SCHEDULER_API_KEY is required")
	}

	client, err := scheduler.NewClient(baseURL, scheduler.WithAPIKey(apiKey))
	if err != nil {
		log.Fatalf("new sdk client: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	fmt.Printf("SDK target: %s\n", baseURL)

	// 1) List jobs.
	jobs, err := client.ListJobs(ctx)
	if err != nil {
		log.Fatalf("list jobs: %v", err)
	}
	fmt.Printf("Jobs: %d\n", len(jobs))

	// 2) Optionally create a demo job.
	createdJobID := ""
	created, createErr := client.CreateJob(ctx, scheduler.CreateJobRequest{
		Name:   fmt.Sprintf("SDK Demo %s", time.Now().UTC().Format("20060102-150405")),
		Method: "POST",
		URL:    envOrDefault("SCHEDULER_DEMO_URL", "https://httpbin.org/post"),
		Headers: map[string]string{
			"Content-Type": "application/json",
		},
		Body: map[string]any{
			"source": "sdk-testing-main",
		},
		CronExpression: envOrDefault("SCHEDULER_DEMO_CRON", "*/15 * * * *"),
		Timezone:       envOrDefault("SCHEDULER_DEMO_TIMEZONE", "UTC"),
		RetryMax:       1,
		TimeoutSeconds: 30,
		Enabled:        true,
	})
	if createErr != nil {
		log.Fatalf("create demo job: %v", createErr)
	}
	createdJobID = created.ID
	fmt.Printf("Created demo job: %s (%s)\n", created.ID, created.Name)

	// 3) Pick a job to inspect/trigger.
	targetJobID := strings.TrimSpace(os.Getenv("SCHEDULER_JOB_ID"))
	if targetJobID == "" {
		if createdJobID != "" {
			targetJobID = createdJobID
		} else if len(jobs) > 0 {
			targetJobID = jobs[0].ID
		}
	}

	if targetJobID != "" {
		job, getErr := client.GetJob(ctx, targetJobID)
		if getErr != nil {
			log.Fatalf("get job %s: %v", targetJobID, getErr)
		}
		fmt.Printf("Selected job: %s [%s] %s\n", job.ID, job.Method, job.URL)

		if triggerErr := client.TriggerJob(ctx, targetJobID); triggerErr != nil {
			log.Fatalf("trigger job %s: %v", targetJobID, triggerErr)
		}
		fmt.Printf("Triggered job: %s\n", targetJobID)

		runs, runsErr := client.ListJobRuns(ctx, targetJobID)
		if runsErr != nil {
			log.Fatalf("list runs for job %s: %v", targetJobID, runsErr)
		}
		fmt.Printf("Runs for selected job: %d\n", len(runs))
		if len(runs) > 0 {
			run, runErr := client.GetRun(ctx, runs[0].ID)
			if runErr != nil {
				log.Fatalf("get run %s: %v", runs[0].ID, runErr)
			}
			fmt.Printf("Latest run: %s status=%s attempts=%d\n", run.ID, run.Status, run.AttemptNumber)
		}
	} else {
		fmt.Println("No job available. Set SCHEDULER_CREATE_DEMO_JOB=true or SCHEDULER_JOB_ID.")
	}

	// 4) List API keys.
	keys, err := client.ListAPIKeys(ctx)
	if err != nil {
		log.Fatalf("list api keys: %v", err)
	}
	fmt.Printf("API Keys: %d\n", len(keys))

	fmt.Println("SDK smoke test completed.")
}

func envOrDefault(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
