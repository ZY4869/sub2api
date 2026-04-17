ALTER TABLE ops_request_traces
    ADD COLUMN IF NOT EXISTS upstream_path VARCHAR(255) NOT NULL DEFAULT '';

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_upstream_path_time
    ON ops_request_traces (upstream_path, created_at DESC)
    WHERE upstream_path <> '';
