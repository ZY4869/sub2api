ALTER TABLE scheduled_test_plans
    ADD COLUMN IF NOT EXISTS model_input_mode VARCHAR(16) NOT NULL DEFAULT 'catalog',
    ADD COLUMN IF NOT EXISTS manual_model_id VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS request_alias VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN IF NOT EXISTS source_protocol VARCHAR(32) NOT NULL DEFAULT '';

COMMENT ON COLUMN scheduled_test_plans.model_input_mode IS 'Model selection mode: catalog or manual';
COMMENT ON COLUMN scheduled_test_plans.manual_model_id IS 'Manual model ID used when model_input_mode=manual';
COMMENT ON COLUMN scheduled_test_plans.request_alias IS 'Optional request alias for manual model execution';
COMMENT ON COLUMN scheduled_test_plans.source_protocol IS 'Optional protocol gateway source protocol for scheduled tests';
