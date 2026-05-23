-- 120_add_channel_monitor_openai_api_mode.sql
-- Persist OpenAI channel monitor protocol mode while keeping chat completions as the safe default.

ALTER TABLE channel_monitors
	ADD COLUMN IF NOT EXISTS openai_api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

ALTER TABLE channel_monitor_request_templates
	ADD COLUMN IF NOT EXISTS openai_api_mode VARCHAR(32) NOT NULL DEFAULT 'chat_completions';

DO $$
BEGIN
	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conname = 'channel_monitors_openai_api_mode_check'
	) THEN
		ALTER TABLE channel_monitors
			ADD CONSTRAINT channel_monitors_openai_api_mode_check
			CHECK (
				openai_api_mode IN ('chat_completions', 'responses')
				AND (provider = 'openai' OR openai_api_mode = 'chat_completions')
			);
	END IF;

	IF NOT EXISTS (
		SELECT 1 FROM pg_constraint
		WHERE conname = 'channel_monitor_request_templates_openai_api_mode_check'
	) THEN
		ALTER TABLE channel_monitor_request_templates
			ADD CONSTRAINT channel_monitor_request_templates_openai_api_mode_check
			CHECK (
				openai_api_mode IN ('chat_completions', 'responses')
				AND (provider = 'openai' OR openai_api_mode = 'chat_completions')
			);
	END IF;
END $$;
