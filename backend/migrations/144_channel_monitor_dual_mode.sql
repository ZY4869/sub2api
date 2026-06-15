-- 144_channel_monitor_dual_mode.sql
-- Adds direct/account-pool probe mode support for channel monitors.

ALTER TABLE channel_monitors
	DROP CONSTRAINT IF EXISTS channel_monitors_provider_check;

ALTER TABLE channel_monitor_request_templates
	DROP CONSTRAINT IF EXISTS channel_monitor_request_templates_provider_check;

ALTER TABLE channel_monitors
	DROP CONSTRAINT IF EXISTS channel_monitors_openai_api_mode_check;

ALTER TABLE channel_monitor_request_templates
	DROP CONSTRAINT IF EXISTS channel_monitor_request_templates_openai_api_mode_check;

ALTER TABLE channel_monitors
	ADD COLUMN IF NOT EXISTS probe_mode VARCHAR(24) NOT NULL DEFAULT 'direct',
	ADD COLUMN IF NOT EXISTS request_protocol VARCHAR(24) NOT NULL DEFAULT 'openai',
	ADD COLUMN IF NOT EXISTS account_ids BIGINT[] NOT NULL DEFAULT '{}'::bigint[],
	ADD COLUMN IF NOT EXISTS model_source_protocols JSONB NOT NULL DEFAULT '{}'::jsonb,
	ADD COLUMN IF NOT EXISTS model_probe_strategy VARCHAR(24) NOT NULL DEFAULT 'all_selected',
	ADD COLUMN IF NOT EXISTS test_prompt_template TEXT NOT NULL DEFAULT '';

ALTER TABLE channel_monitor_request_templates
	ADD COLUMN IF NOT EXISTS request_protocol VARCHAR(24) NOT NULL DEFAULT 'openai',
	ADD COLUMN IF NOT EXISTS test_prompt_template TEXT NOT NULL DEFAULT '';

ALTER TABLE channel_monitor_histories
	ADD COLUMN IF NOT EXISTS account_id BIGINT,
	ADD COLUMN IF NOT EXISTS account_name_snapshot TEXT NOT NULL DEFAULT '',
	ADD COLUMN IF NOT EXISTS probe_mode VARCHAR(24) NOT NULL DEFAULT 'direct';

UPDATE channel_monitors
SET request_protocol = CASE
	WHEN provider IN ('anthropic', 'antigravity') THEN 'anthropic'
	WHEN provider IN ('gemini', 'google') THEN 'gemini'
	ELSE 'openai'
END
WHERE request_protocol = 'openai';

UPDATE channel_monitor_request_templates
SET request_protocol = CASE
	WHEN provider IN ('anthropic', 'antigravity') THEN 'anthropic'
	WHEN provider IN ('gemini', 'google') THEN 'gemini'
	ELSE 'openai'
END
WHERE request_protocol = 'openai';

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'channel_monitors'::regclass
		  AND conname = 'channel_monitors_probe_mode_check'
	) THEN
		ALTER TABLE channel_monitors
			ADD CONSTRAINT channel_monitors_probe_mode_check
			CHECK (probe_mode IN ('direct', 'account_pool'));
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'channel_monitors'::regclass
		  AND conname = 'channel_monitors_request_protocol_check'
	) THEN
		ALTER TABLE channel_monitors
			ADD CONSTRAINT channel_monitors_request_protocol_check
			CHECK (request_protocol IN ('openai', 'anthropic', 'gemini'));
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'channel_monitors'::regclass
		  AND conname = 'channel_monitors_model_probe_strategy_check'
	) THEN
		ALTER TABLE channel_monitors
			ADD CONSTRAINT channel_monitors_model_probe_strategy_check
			CHECK (model_probe_strategy IN ('primary_only', 'all_selected'));
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'channel_monitors'::regclass
		  AND conname = 'channel_monitors_openai_api_mode_check'
	) THEN
		ALTER TABLE channel_monitors
			ADD CONSTRAINT channel_monitors_openai_api_mode_check
			CHECK (
				openai_api_mode IN ('chat_completions', 'responses')
				AND (request_protocol = 'openai' OR openai_api_mode = 'chat_completions')
			);
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'channel_monitor_request_templates'::regclass
		  AND conname = 'channel_monitor_request_templates_request_protocol_check'
	) THEN
		ALTER TABLE channel_monitor_request_templates
			ADD CONSTRAINT channel_monitor_request_templates_request_protocol_check
			CHECK (request_protocol IN ('openai', 'anthropic', 'gemini'));
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'channel_monitor_request_templates'::regclass
		  AND conname = 'channel_monitor_request_templates_openai_api_mode_check'
	) THEN
		ALTER TABLE channel_monitor_request_templates
			ADD CONSTRAINT channel_monitor_request_templates_openai_api_mode_check
			CHECK (
				openai_api_mode IN ('chat_completions', 'responses')
				AND (request_protocol = 'openai' OR openai_api_mode = 'chat_completions')
			);
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conrelid = 'channel_monitor_histories'::regclass
		  AND conname = 'channel_monitor_histories_probe_mode_check'
	) THEN
		ALTER TABLE channel_monitor_histories
			ADD CONSTRAINT channel_monitor_histories_probe_mode_check
			CHECK (probe_mode IN ('direct', 'account_pool'));
	END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_channel_monitor_histories_account_created_at
	ON channel_monitor_histories(account_id, created_at DESC)
	WHERE account_id IS NOT NULL;
