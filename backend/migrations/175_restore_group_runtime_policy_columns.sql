-- The repository runner executes the entire file as forward SQL; Goose Down
-- sections in migrations 173/174 removed their own additions. Preserve those
-- checksummed migrations and restore the columns through this additive repair.
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS max_reasoning_effort_over_limit VARCHAR(20) NOT NULL DEFAULT 'downgrade',
    ADD COLUMN IF NOT EXISTS force_openai_fast BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS free_openai_fast BOOLEAN NOT NULL DEFAULT FALSE;

UPDATE groups
SET max_reasoning_effort_over_limit = 'downgrade'
WHERE COALESCE(TRIM(max_reasoning_effort_over_limit), '') = '';

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint
        WHERE conname = 'groups_max_reasoning_effort_over_limit_check'
          AND conrelid = 'groups'::regclass
    ) THEN
        ALTER TABLE groups
            ADD CONSTRAINT groups_max_reasoning_effort_over_limit_check
            CHECK (max_reasoning_effort_over_limit IN ('downgrade', 'deny'));
    END IF;
END $$;
