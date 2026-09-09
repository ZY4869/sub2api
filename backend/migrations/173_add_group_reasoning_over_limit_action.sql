-- +goose Up
-- +goose StatementBegin
-- Add the explicit action for group reasoning limits.
-- Empty/legacy rows default to downgrade to preserve existing request behavior.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS max_reasoning_effort_over_limit VARCHAR(20) NOT NULL DEFAULT 'downgrade';

UPDATE groups
SET max_reasoning_effort_over_limit = 'downgrade'
WHERE COALESCE(TRIM(max_reasoning_effort_over_limit), '') = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'groups_max_reasoning_effort_over_limit_check'
    ) THEN
        ALTER TABLE groups
            ADD CONSTRAINT groups_max_reasoning_effort_over_limit_check
            CHECK (max_reasoning_effort_over_limit IN ('downgrade', 'deny'));
    END IF;
END $$;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
-- Rollback removes only the column and its check constraint introduced above.
-- Existing group reasoning limits and request behavior are otherwise unchanged.
ALTER TABLE groups
    DROP CONSTRAINT IF EXISTS groups_max_reasoning_effort_over_limit_check;

ALTER TABLE groups
    DROP COLUMN IF EXISTS max_reasoning_effort_over_limit;
-- +goose StatementEnd
