# Scheduler Go SDK

Import path:

```go
import "github.com/brian-nunez/btick/sdk/go/scheduler"
```

## Create Client

```go
client, err := scheduler.NewClient(
    "http://localhost:8080",
    scheduler.WithAPIKey("skd_xxxxx_xxxxx"),
)
if err != nil {
    panic(err)
}
```

You can also set/update auth later:

```go
client.SetAPIKey("skd_xxxxx_xxxxx")
```

## Example Flow

```go
ctx := context.Background()

created, err := client.CreateJob(ctx, scheduler.CreateJobRequest{
    Name:   "Nightly sync",
    Method: "POST",
    URL:    "https://api.example.com/sync",
    Headers: map[string]string{
        "Content-Type": "application/json",
    },
    Body: map[string]any{
        "source": "scheduler",
    },
    CronExpression: "0 2 * * *",
    Timezone:       "America/Phoenix",
    RetryMax:       3,
    TimeoutSeconds: 60,
    Enabled:        true,
})
if err != nil {
    panic(err)
}

_ = client.TriggerJob(ctx, created.ID)
jobs, _ := client.ListJobs(ctx)
fmt.Println(len(jobs))
```

## Supported Methods

- `CreateJob`
- `ListJobs`
- `GetJob`
- `UpdateJob`
- `DeleteJob`
- `PauseJob`
- `ResumeJob`
- `TriggerJob`
- `ListJobRuns`
- `GetRun`
- `CreateAPIKey`
- `ListAPIKeys`
- `RevokeAPIKey`
