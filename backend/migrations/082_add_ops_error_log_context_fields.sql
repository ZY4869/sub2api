-- Add richer ops error context for endpoint/model/request_type observability.
ALTER TABLE ops_error_logs
    ADD COLUMN IF NOT EXISTS inbound_endpoint VARCHAR(128),
    ADD COLUMN IF NOT EXISTS upstream_endpoint VARCHAR(128),
    ADD COLUMN IF NOT EXISTS requested_model VARCHAR(100),
    ADD COLUMN IF NOT EXISTS upstream_model VARCHAR(100),
    ADD COLUMN IF NOT EXISTS request_type SMALLINT,
    ADD COLUMN IF NOT EXISTS upstream_url TEXT;

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1
        FROM pg_constraint
        WHERE conname = 'ops_error_logs_request_type_check'
    ) THEN
        ALTER TABLE ops_error_logs
            ADD CONSTRAINT ops_error_logs_request_type_check
            CHECK (request_type IS NULL OR request_type IN (1, 2, 3));
    END IF;
END $$;
