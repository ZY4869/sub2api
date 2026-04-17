CREATE TABLE IF NOT EXISTS document_ai_jobs (
    id BIGSERIAL PRIMARY KEY,
    job_id TEXT NOT NULL UNIQUE,
    provider_job_id TEXT NULL,
    provider_batch_id TEXT NULL,
    account_id BIGINT NULL REFERENCES accounts(id) ON DELETE SET NULL,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    group_id BIGINT NULL REFERENCES groups(id) ON DELETE SET NULL,
    mode TEXT NOT NULL,
    model TEXT NOT NULL,
    source_type TEXT NOT NULL,
    file_name TEXT NULL,
    content_type TEXT NULL,
    file_size BIGINT NULL,
    file_hash TEXT NULL,
    status TEXT NOT NULL,
    provider_result_json JSONB NULL,
    normalized_result_json JSONB NULL,
    error_code TEXT NULL,
    error_message TEXT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ NULL,
    last_polled_at TIMESTAMPTZ NULL
);

CREATE INDEX IF NOT EXISTS idx_document_ai_jobs_user_created_at
    ON document_ai_jobs(user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_document_ai_jobs_status_created_at
    ON document_ai_jobs(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_document_ai_jobs_api_key_created_at
    ON document_ai_jobs(api_key_id, created_at DESC);
