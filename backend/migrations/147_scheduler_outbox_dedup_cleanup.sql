ALTER TABLE scheduler_outbox
    ADD COLUMN IF NOT EXISTS dedup_key TEXT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_scheduler_outbox_dedup_key
    ON scheduler_outbox (dedup_key)
    WHERE dedup_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_scheduler_outbox_id_created_at
    ON scheduler_outbox (id, created_at);
