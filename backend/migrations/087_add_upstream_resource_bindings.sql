CREATE TABLE IF NOT EXISTS upstream_resource_bindings (
    id BIGSERIAL PRIMARY KEY,
    resource_kind VARCHAR(64) NOT NULL,
    resource_name TEXT NOT NULL,
    provider_family VARCHAR(32) NOT NULL,
    account_id BIGINT NOT NULL,
    api_key_id BIGINT,
    group_id BIGINT,
    user_id BIGINT,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_upstream_resource_bindings_kind_name
    ON upstream_resource_bindings (resource_kind, resource_name);

CREATE INDEX IF NOT EXISTS idx_upstream_resource_bindings_account_id
    ON upstream_resource_bindings (account_id)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_upstream_resource_bindings_provider_family
    ON upstream_resource_bindings (provider_family)
    WHERE deleted_at IS NULL;
