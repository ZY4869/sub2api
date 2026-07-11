-- Add scan, isolation, idempotency, and cleanup indexes for image batches.

CREATE INDEX IF NOT EXISTS idx_image_batch_jobs_user_created
    ON image_batch_jobs (user_id, created_at DESC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_image_batch_jobs_status_scan
    ON image_batch_jobs (status, updated_at ASC)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_image_batch_jobs_api_key_idempotency
    ON image_batch_jobs (api_key_id, idempotency_key_hash)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_image_batch_jobs_cleanup
    ON image_batch_jobs (outputs_deleted_at, completed_at)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_image_batch_items_job_status
    ON image_batch_items (job_id, status, id);

CREATE INDEX IF NOT EXISTS idx_image_batch_outputs_job_created
    ON image_batch_outputs (job_id, created_at ASC)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_image_batch_outputs_item
    ON image_batch_outputs (item_id)
    WHERE deleted_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_image_batch_outputs_job_custom_sha
    ON image_batch_outputs (job_id, custom_id, sha256)
    WHERE deleted_at IS NULL AND sha256 <> '';
