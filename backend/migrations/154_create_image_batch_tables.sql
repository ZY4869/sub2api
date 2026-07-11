-- Create local image batch job tables.

CREATE TABLE IF NOT EXISTS image_batch_jobs (
    id UUID PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id),
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id),
    group_id BIGINT REFERENCES groups(id),
    provider TEXT NOT NULL,
    display_model_id TEXT NOT NULL,
    target_model_id TEXT NOT NULL DEFAULT '',
    size TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN (
        'created',
        'uploading',
        'submitted',
        'running',
        'indexing',
        'settling',
        'completed',
        'failed',
        'cancelled',
        'output_deleted'
    )),
    item_count INTEGER NOT NULL DEFAULT 0,
    success_count INTEGER NOT NULL DEFAULT 0,
    failed_count INTEGER NOT NULL DEFAULT 0,
    cancelled_count INTEGER NOT NULL DEFAULT 0,
    hold_request_id TEXT NOT NULL DEFAULT '',
    idempotency_key_hash TEXT NOT NULL,
    request_fingerprint TEXT NOT NULL,
    provider_batch_name TEXT NOT NULL DEFAULT '',
    provider_result_file_name TEXT NOT NULL DEFAULT '',
    friendly_error TEXT NOT NULL DEFAULT '',
    error_id TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    submitted_at TIMESTAMPTZ,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    outputs_deleted_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS image_batch_items (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES image_batch_jobs(id) ON DELETE CASCADE,
    custom_id TEXT NOT NULL,
    prompt TEXT NOT NULL,
    n INTEGER NOT NULL DEFAULT 1,
    size TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('pending', 'success', 'failed', 'cancelled')),
    output_count INTEGER NOT NULL DEFAULT 0,
    friendly_error TEXT NOT NULL DEFAULT '',
    error_id TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, custom_id)
);

CREATE TABLE IF NOT EXISTS image_batch_events (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES image_batch_jobs(id) ON DELETE CASCADE,
    item_id BIGINT REFERENCES image_batch_items(id) ON DELETE SET NULL,
    event_type TEXT NOT NULL,
    status_from TEXT NOT NULL DEFAULT '',
    status_to TEXT NOT NULL DEFAULT '',
    message TEXT NOT NULL DEFAULT '',
    request_id TEXT NOT NULL DEFAULT '',
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS image_batch_outputs (
    id BIGSERIAL PRIMARY KEY,
    job_id UUID NOT NULL REFERENCES image_batch_jobs(id) ON DELETE CASCADE,
    item_id BIGINT NOT NULL REFERENCES image_batch_items(id) ON DELETE CASCADE,
    custom_id TEXT NOT NULL,
    content_type TEXT NOT NULL DEFAULT 'image/png',
    storage_backend TEXT NOT NULL DEFAULT 'db',
    content BYTEA,
    size_bytes BIGINT NOT NULL DEFAULT 0,
    sha256 TEXT NOT NULL DEFAULT '',
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);
