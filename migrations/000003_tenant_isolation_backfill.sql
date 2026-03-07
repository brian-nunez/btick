-- Assign each existing user a deterministic tenant when missing.
UPDATE users
SET tenant_id = id,
    updated_at = now()
WHERE tenant_id IS NULL;

-- Backfill job tenant ownership from creator when possible.
UPDATE jobs j
SET tenant_id = u.tenant_id,
    updated_at = now()
FROM users u
WHERE j.tenant_id IS NULL
  AND j.created_by = u.id::text
  AND u.tenant_id IS NOT NULL;

-- Backfill API key tenant ownership from creator when possible.
UPDATE api_keys k
SET tenant_id = u.tenant_id
FROM users u
WHERE k.tenant_id IS NULL
  AND k.created_by = u.id::text
  AND u.tenant_id IS NOT NULL;
