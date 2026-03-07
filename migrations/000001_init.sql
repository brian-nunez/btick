CREATE TABLE IF NOT EXISTS jobs (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,

    method TEXT NOT NULL,
    url TEXT NOT NULL,

    headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    body JSONB,

    cron_expression TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'UTC',

    retry_max INT NOT NULL DEFAULT 0,
    timeout_seconds INT NOT NULL DEFAULT 60,

    enabled BOOLEAN NOT NULL DEFAULT true,

    next_run_at TIMESTAMPTZ,
    last_run_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    claimed_at TIMESTAMPTZ,
    claimed_by TEXT,

    tenant_id UUID,
    created_by TEXT,

    CONSTRAINT jobs_retry_max_check CHECK (retry_max >= 0),
    CONSTRAINT jobs_timeout_seconds_check CHECK (timeout_seconds > 0)
);

CREATE TABLE IF NOT EXISTS job_runs (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,

    trigger_type TEXT NOT NULL DEFAULT 'scheduled',
    attempt_number INT NOT NULL,
    status TEXT NOT NULL,

    request_method TEXT NOT NULL,
    request_url TEXT NOT NULL,
    request_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
    request_body JSONB,

    response_status_code INT,
    response_headers JSONB,
    response_body TEXT,

    error_message TEXT,

    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL,

    key_prefix TEXT NOT NULL,
    key_hash TEXT NOT NULL,

    scopes JSONB NOT NULL DEFAULT '[]'::jsonb,

    tenant_id UUID,
    created_by TEXT,

    last_used_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS manual_job_triggers (
    id UUID PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES jobs(id) ON DELETE CASCADE,
    triggered_by TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    claimed_at TIMESTAMPTZ,
    claimed_by TEXT,
    completed_at TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_jobs_next_run_at
ON jobs (next_run_at)
WHERE enabled = true;

CREATE INDEX IF NOT EXISTS idx_jobs_claimed_at
ON jobs (claimed_at);

CREATE INDEX IF NOT EXISTS idx_job_runs_job_id_started_at
ON job_runs (job_id, started_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_keys_key_prefix
ON api_keys (key_prefix);

CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id
ON api_keys (tenant_id);

CREATE INDEX IF NOT EXISTS idx_api_keys_revoked_at
ON api_keys (revoked_at);

CREATE INDEX IF NOT EXISTS idx_manual_job_triggers_job_id
ON manual_job_triggers (job_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_manual_job_triggers_claimed_at
ON manual_job_triggers (claimed_at);
