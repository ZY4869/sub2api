CREATE TABLE IF NOT EXISTS content_moderation_audits (
  id BIGSERIAL PRIMARY KEY,
  request_id TEXT NOT NULL DEFAULT '',
  client_request_id TEXT NOT NULL DEFAULT '',
  user_id BIGINT,
  api_key_id BIGINT,
  provider TEXT NOT NULL DEFAULT '',
  model TEXT NOT NULL DEFAULT '',
  source_endpoint TEXT NOT NULL DEFAULT '',
  content_hash TEXT NOT NULL,
  content_summary TEXT NOT NULL DEFAULT '',
  hit BOOLEAN NOT NULL DEFAULT FALSE,
  dedupe_hit BOOLEAN NOT NULL DEFAULT FALSE,
  error_reason TEXT NOT NULL DEFAULT '',
  latency_ms INTEGER NOT NULL DEFAULT 0,
  created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_content_moderation_audits_created_at
  ON content_moderation_audits (created_at DESC, id DESC);

CREATE INDEX IF NOT EXISTS idx_content_moderation_audits_content_hash_created_at
  ON content_moderation_audits (content_hash, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_content_moderation_audits_request_id
  ON content_moderation_audits (request_id);

CREATE INDEX IF NOT EXISTS idx_content_moderation_audits_user_id
  ON content_moderation_audits (user_id);
