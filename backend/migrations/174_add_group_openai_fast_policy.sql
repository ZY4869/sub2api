-- +goose Up
-- +goose StatementBegin
ALTER TABLE groups
    ADD COLUMN IF NOT EXISTS force_openai_fast BOOLEAN NOT NULL DEFAULT FALSE,
    ADD COLUMN IF NOT EXISTS free_openai_fast BOOLEAN NOT NULL DEFAULT FALSE;
-- +goose StatementEnd

-- +goose Down
-- +goose StatementBegin
ALTER TABLE groups
    DROP COLUMN IF EXISTS force_openai_fast,
    DROP COLUMN IF EXISTS free_openai_fast;
-- +goose StatementEnd
