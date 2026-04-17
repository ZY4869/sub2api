CREATE TABLE IF NOT EXISTS usage_repair_tasks (
    id BIGSERIAL PRIMARY KEY,
    kind VARCHAR(64) NOT NULL,
    days INTEGER NOT NULL DEFAULT 30,
    status VARCHAR(32) NOT NULL DEFAULT 'pending',
    created_by BIGINT NOT NULL,
    processed_rows BIGINT NOT NULL DEFAULT 0,
    repaired_rows BIGINT NOT NULL DEFAULT 0,
    skipped_rows BIGINT NOT NULL DEFAULT 0,
    error_message TEXT NULL,
    canceled_by BIGINT NULL,
    canceled_at TIMESTAMPTZ NULL,
    started_at TIMESTAMPTZ NULL,
    finished_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_usage_repair_tasks_status_created_at
    ON usage_repair_tasks(status, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_usage_repair_tasks_kind_created_at
    ON usage_repair_tasks(kind, created_at DESC);
