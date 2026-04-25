-- 105_add_channel_monitor_request_templates.sql
-- Request templates that can be applied to monitors (headers/body override modes).

CREATE TABLE IF NOT EXISTS channel_monitor_request_templates (
	id BIGSERIAL PRIMARY KEY,
	name VARCHAR(100) NOT NULL,
	provider VARCHAR(32) NOT NULL,
	description TEXT,
	extra_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
	body_override_mode VARCHAR(20) NOT NULL DEFAULT 'off',
	body_override JSONB NOT NULL DEFAULT '{}'::jsonb,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	CONSTRAINT channel_monitor_request_templates_unique_name UNIQUE (name),
	CONSTRAINT channel_monitor_request_templates_provider_check CHECK (provider IN ('openai', 'anthropic', 'gemini', 'grok', 'antigravity')),
	CONSTRAINT channel_monitor_request_templates_override_mode_check CHECK (body_override_mode IN ('off', 'merge', 'replace'))
);

ALTER TABLE channel_monitors
	ADD COLUMN IF NOT EXISTS template_id BIGINT REFERENCES channel_monitor_request_templates(id) ON DELETE SET NULL,
	ADD COLUMN IF NOT EXISTS extra_headers JSONB NOT NULL DEFAULT '{}'::jsonb,
	ADD COLUMN IF NOT EXISTS body_override_mode VARCHAR(20) NOT NULL DEFAULT 'off',
	ADD COLUMN IF NOT EXISTS body_override JSONB NOT NULL DEFAULT '{}'::jsonb;

CREATE INDEX IF NOT EXISTS idx_channel_monitors_template_id
	ON channel_monitors(template_id, id);

-- Ensure override mode check exists for channel_monitors (idempotent).
DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1
		FROM pg_constraint
		WHERE conrelid = 'channel_monitors'::regclass
		  AND conname = 'channel_monitors_body_override_mode_check'
	) THEN
		ALTER TABLE channel_monitors
			ADD CONSTRAINT channel_monitors_body_override_mode_check
			CHECK (body_override_mode IN ('off', 'merge', 'replace'));
	END IF;
END
$$;
