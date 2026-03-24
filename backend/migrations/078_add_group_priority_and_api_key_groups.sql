ALTER TABLE groups
ADD COLUMN IF NOT EXISTS priority INT NOT NULL DEFAULT 1;

CREATE INDEX IF NOT EXISTS idx_groups_priority ON groups (priority);

CREATE TABLE IF NOT EXISTS api_key_groups (
    api_key_id BIGINT NOT NULL REFERENCES api_keys(id) ON DELETE CASCADE,
    group_id BIGINT NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    quota DECIMAL(20,8) NOT NULL DEFAULT 0,
    quota_used DECIMAL(20,8) NOT NULL DEFAULT 0,
    model_patterns JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (api_key_id, group_id)
);

CREATE INDEX IF NOT EXISTS idx_api_key_groups_group_id ON api_key_groups (group_id);
CREATE INDEX IF NOT EXISTS idx_api_key_groups_api_key_id ON api_key_groups (api_key_id);

INSERT INTO api_key_groups (api_key_id, group_id, quota, quota_used, model_patterns, created_at, updated_at)
SELECT
    ak.id,
    ak.group_id,
    0,
    0,
    '[]'::jsonb,
    NOW(),
    NOW()
FROM api_keys ak
WHERE ak.group_id IS NOT NULL
  AND ak.deleted_at IS NULL
ON CONFLICT (api_key_id, group_id) DO NOTHING;
