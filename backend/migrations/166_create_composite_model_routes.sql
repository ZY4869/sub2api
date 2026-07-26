CREATE TABLE IF NOT EXISTS composite_model_routes (
    id BIGSERIAL PRIMARY KEY,
    parent_group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    display_model_id VARCHAR(191) NOT NULL,
    target_group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE RESTRICT,
    target_model_id VARCHAR(191) NOT NULL DEFAULT '',
    priority INT NOT NULL DEFAULT 50,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    notes TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT ck_composite_model_routes_no_self_target CHECK (parent_group_id <> target_group_id)
);

CREATE INDEX IF NOT EXISTS idx_composite_model_routes_parent_group_id
    ON composite_model_routes (parent_group_id);

CREATE INDEX IF NOT EXISTS idx_composite_model_routes_target_group_id
    ON composite_model_routes (target_group_id);

CREATE INDEX IF NOT EXISTS idx_composite_model_routes_parent_display_enabled_priority
    ON composite_model_routes (parent_group_id, display_model_id, enabled, priority);
