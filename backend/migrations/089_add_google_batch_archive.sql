CREATE TABLE IF NOT EXISTS google_batch_archive_jobs (
    id BIGSERIAL PRIMARY KEY,
    public_batch_name VARCHAR(255) NOT NULL,
    public_protocol VARCHAR(32) NOT NULL DEFAULT 'ai_studio',
    execution_provider_family VARCHAR(32) NOT NULL DEFAULT 'ai_studio',
    execution_batch_name VARCHAR(255) NOT NULL,
    source_account_id BIGINT NOT NULL,
    execution_account_id BIGINT NOT NULL,
    api_key_id BIGINT NULL,
    group_id BIGINT NULL,
    user_id BIGINT NULL,
    requested_model VARCHAR(255) NOT NULL DEFAULT '',
    conversion_direction VARCHAR(64) NOT NULL DEFAULT 'none',
    state VARCHAR(64) NOT NULL DEFAULT 'created',
    official_expires_at TIMESTAMPTZ NULL,
    prefetch_due_at TIMESTAMPTZ NULL,
    last_public_result_access_at TIMESTAMPTZ NULL,
    next_poll_at TIMESTAMPTZ NULL,
    poll_attempts INTEGER NOT NULL DEFAULT 0,
    archive_state VARCHAR(64) NOT NULL DEFAULT 'pending',
    billing_settlement_state VARCHAR(64) NOT NULL DEFAULT 'pending',
    retention_expires_at TIMESTAMPTZ NULL,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_google_batch_archive_jobs_public_batch_name_live
    ON google_batch_archive_jobs (public_batch_name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_archive_jobs_execution_batch_name_live
    ON google_batch_archive_jobs (execution_batch_name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_archive_jobs_next_poll_at_live
    ON google_batch_archive_jobs (next_poll_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_archive_jobs_prefetch_due_at_live
    ON google_batch_archive_jobs (prefetch_due_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_archive_jobs_retention_expires_at_live
    ON google_batch_archive_jobs (retention_expires_at)
    WHERE deleted_at IS NULL;

CREATE TABLE IF NOT EXISTS google_batch_archive_objects (
    id BIGSERIAL PRIMARY KEY,
    job_id BIGINT NOT NULL REFERENCES google_batch_archive_jobs(id) ON DELETE CASCADE,
    public_resource_kind VARCHAR(32) NOT NULL,
    public_resource_name VARCHAR(255) NOT NULL,
    execution_resource_name VARCHAR(255) NOT NULL DEFAULT '',
    storage_backend VARCHAR(32) NOT NULL DEFAULT 'local_fs',
    relative_path VARCHAR(1024) NOT NULL DEFAULT '',
    content_type VARCHAR(255) NOT NULL DEFAULT '',
    size_bytes BIGINT NOT NULL DEFAULT 0,
    sha256 VARCHAR(128) NOT NULL DEFAULT '',
    is_result_payload BOOLEAN NOT NULL DEFAULT FALSE,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_google_batch_archive_objects_public_resource_live
    ON google_batch_archive_objects (public_resource_kind, public_resource_name)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_archive_objects_job_id_live
    ON google_batch_archive_objects (job_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_google_batch_archive_objects_result_payload_live
    ON google_batch_archive_objects (is_result_payload)
    WHERE deleted_at IS NULL;

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS operation_type VARCHAR(64) NULL;

ALTER TABLE usage_logs
    ADD COLUMN IF NOT EXISTS charge_source VARCHAR(64) NULL;
