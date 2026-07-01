-- 152_allow_grok_channel_monitor_request_protocol.sql
-- Allows Grok protocol probes in channel monitors and request templates.

ALTER TABLE channel_monitors
	DROP CONSTRAINT IF EXISTS channel_monitors_request_protocol_check;

ALTER TABLE channel_monitors
	ADD CONSTRAINT channel_monitors_request_protocol_check
	CHECK (request_protocol IN ('openai', 'anthropic', 'gemini', 'grok'));

ALTER TABLE channel_monitor_request_templates
	DROP CONSTRAINT IF EXISTS channel_monitor_request_templates_request_protocol_check;

ALTER TABLE channel_monitor_request_templates
	ADD CONSTRAINT channel_monitor_request_templates_request_protocol_check
	CHECK (request_protocol IN ('openai', 'anthropic', 'gemini', 'grok'));
