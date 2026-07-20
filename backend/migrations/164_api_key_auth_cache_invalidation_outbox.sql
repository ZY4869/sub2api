CREATE TABLE IF NOT EXISTS api_key_auth_cache_invalidation_outbox (
    id BIGSERIAL PRIMARY KEY,
    event_type TEXT NOT NULL,
    cache_key TEXT NULL,
    user_id BIGINT NULL,
    group_id BIGINT NULL,
    dedup_key TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_api_key_auth_cache_invalidation_outbox_target CHECK (
        (
            event_type = 'api_key_auth_cache_invalidation_key'
            AND cache_key IS NOT NULL
            AND user_id IS NULL
            AND group_id IS NULL
        )
        OR (
            event_type = 'api_key_auth_cache_invalidation_user'
            AND cache_key IS NULL
            AND user_id IS NOT NULL
            AND group_id IS NULL
        )
        OR (
            event_type = 'api_key_auth_cache_invalidation_group'
            AND cache_key IS NULL
            AND user_id IS NULL
            AND group_id IS NOT NULL
        )
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_api_key_auth_cache_invalidation_outbox_dedup_key
    ON api_key_auth_cache_invalidation_outbox (dedup_key);

CREATE INDEX IF NOT EXISTS idx_api_key_auth_cache_invalidation_outbox_id_created_at
    ON api_key_auth_cache_invalidation_outbox (id, created_at);
