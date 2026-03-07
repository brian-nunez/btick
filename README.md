# BTick Scheduler Service

BTick is a generic API-driven scheduler service for recurring HTTP jobs.

It provides:

- Scheduler API (`cmd/scheduler-api`):
  - JSON API under `/api/v1`
  - server-rendered UI with `templ` + `templui` + `tailwindcss` + `htmx`
  - API key management and API key auth
- Scheduler Worker (`cmd/scheduler-worker`):
  - Postgres polling for due jobs
  - safe multi-worker claim semantics
  - HTTP execution with retries and run history persistence
  - execution through `task-orchestration`

## Architecture

- Control plane: API service
- Execution plane: Worker service
- Source of truth: Postgres (`jobs`, `job_runs`, `api_keys`)
- Scheduling model: persisted `next_run_at` + cron expression
- No in-memory timer dependence

## Required Libraries

- [`github.com/brian-nunez/task-orchestration`](https://github.com/brian-nunez/task-orchestration)
- [`github.com/brian-nunez/baccess`](https://github.com/brian-nunez/baccess)

## Project Layout

```text
cmd/
  scheduler-api/
  scheduler-worker/
internal/
  apikeys/
  auth/
  authorization/
  config/
  db/
  handlers/
    errors/
    v1/
  httpserver/
  jobs/
  runs/
  scheduler/
  worker/
migrations/
queries/
sdk/
  go/
    scheduler/
views/
```

## Environment Variables

Shared:

- `DATABASE_URL` (required)

API:

- `PORT` (default `8080`)
- `STATIC_ASSETS_PATH` (default `./assets`)

Worker:

- `WORKER_ID` (default `scheduler-worker-1`)
- `WORKER_CONCURRENCY` (default `4`)
- `WORKER_POLL_INTERVAL_MS` (default `2000`)
- `WORKER_STALE_CLAIM_MINUTES` (default `5`)
- `WORKER_MAX_CLAIMS_PER_TICK` (default `20`)
- `WORKER_TASK_LOG_PATH` (default `./logs/tasks`)
- `WORKER_TASK_STATE_DB_PATH` (default `./data/task_orchestration.db`)
- `WORKER_HTTP_TIMEOUT_SECONDS` (default `90`)

## Run Locally

1. Set `DATABASE_URL`.
2. Start API:

```bash
go run ./cmd/scheduler-api
```

3. Start worker:

```bash
go run ./cmd/scheduler-worker
```

Migrations run automatically on startup.

## API Endpoints

Health:

- `GET /api/v1/health`
- `GET /healthz`
- `GET /readyz`

Jobs:

- `POST /api/v1/jobs`
- `GET /api/v1/jobs`
- `GET /api/v1/jobs/{jobId}`
- `PATCH /api/v1/jobs/{jobId}`
- `DELETE /api/v1/jobs/{jobId}`
- `POST /api/v1/jobs/{jobId}/pause`
- `POST /api/v1/jobs/{jobId}/resume`
- `POST /api/v1/jobs/{jobId}/trigger`

Runs:

- `GET /api/v1/jobs/{jobId}/runs`
- `GET /api/v1/runs/{runId}`

API Keys:

- `POST /api/v1/api-keys`
- `GET /api/v1/api-keys`
- `POST /api/v1/api-keys/{keyId}/revoke`

## Authentication

Protected API endpoints expect:

```http
Authorization: Bearer <api-key>
```

Keys are generated once and only `key_prefix` + secure hash are stored.

## UI Routes

- `/ui/jobs`
- `/ui/jobs/new`
- `/ui/jobs/{jobId}`
- `/ui/jobs/{jobId}/edit`
- `/ui/jobs/{jobId}/runs`
- `/ui/runs/{runId}`
- `/ui/api-keys`
- `/ui/api-keys/new`

## SDK

Go SDK lives at:

- `sdk/go/scheduler`

See [`sdk/go/scheduler/README.md`](./sdk/go/scheduler/README.md) for examples.

## SQLC + Migrations

- SQL schema migration files: `migrations/`
- Query contracts: `queries/`
- SQLC config: `sqlc.yaml`
- Query implementation package: `internal/db/sqlc`
