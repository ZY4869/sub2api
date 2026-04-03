-- Request traces for protocol gateway troubleshooting.
-- Stores searchable sanitized payloads plus optional encrypted raw payloads.

CREATE TABLE IF NOT EXISTS ops_request_traces (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    request_id VARCHAR(120),
    client_request_id VARCHAR(120),
    upstream_request_id VARCHAR(120),

    user_id BIGINT REFERENCES users(id) ON DELETE SET NULL,
    api_key_id BIGINT REFERENCES api_keys(id) ON DELETE SET NULL,
    account_id BIGINT REFERENCES accounts(id) ON DELETE SET NULL,
    group_id BIGINT REFERENCES groups(id) ON DELETE SET NULL,

    platform VARCHAR(50) NOT NULL DEFAULT '',
    protocol_in VARCHAR(50) NOT NULL DEFAULT '',
    protocol_out VARCHAR(50) NOT NULL DEFAULT '',
    channel VARCHAR(50) NOT NULL DEFAULT '',
    route_path VARCHAR(255) NOT NULL DEFAULT '',
    request_type VARCHAR(50) NOT NULL DEFAULT '',

    requested_model VARCHAR(150) NOT NULL DEFAULT '',
    upstream_model VARCHAR(150) NOT NULL DEFAULT '',
    actual_upstream_model VARCHAR(150) NOT NULL DEFAULT '',

    status VARCHAR(20) NOT NULL DEFAULT '',
    status_code INT NOT NULL DEFAULT 0,
    upstream_status_code INT,
    duration_ms BIGINT NOT NULL DEFAULT 0,
    ttft_ms BIGINT,

    input_tokens INT NOT NULL DEFAULT 0,
    output_tokens INT NOT NULL DEFAULT 0,
    total_tokens INT NOT NULL DEFAULT 0,

    finish_reason VARCHAR(80) NOT NULL DEFAULT '',
    prompt_block_reason VARCHAR(80) NOT NULL DEFAULT '',

    stream BOOLEAN NOT NULL DEFAULT FALSE,
    has_tools BOOLEAN NOT NULL DEFAULT FALSE,
    tool_kinds TEXT[] NOT NULL DEFAULT '{}',
    has_thinking BOOLEAN NOT NULL DEFAULT FALSE,
    thinking_source VARCHAR(80) NOT NULL DEFAULT '',
    thinking_level VARCHAR(40) NOT NULL DEFAULT '',
    thinking_budget INT,
    media_resolution VARCHAR(40) NOT NULL DEFAULT '',
    count_tokens_source VARCHAR(40) NOT NULL DEFAULT '',

    capture_reason VARCHAR(80) NOT NULL DEFAULT '',
    sampled BOOLEAN NOT NULL DEFAULT FALSE,
    raw_available BOOLEAN NOT NULL DEFAULT FALSE,

    inbound_request JSONB,
    normalized_request JSONB,
    upstream_request JSONB,
    upstream_response JSONB,
    gateway_response JSONB,
    tool_trace JSONB,
    request_headers JSONB,
    response_headers JSONB,

    raw_request BYTEA,
    raw_response BYTEA,
    raw_request_bytes INT,
    raw_response_bytes INT,
    raw_request_truncated BOOLEAN NOT NULL DEFAULT FALSE,
    raw_response_truncated BOOLEAN NOT NULL DEFAULT FALSE,

    search_text TEXT NOT NULL DEFAULT ''
);

CREATE TABLE IF NOT EXISTS ops_request_trace_audits (
    id BIGSERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    trace_id BIGINT REFERENCES ops_request_traces(id) ON DELETE SET NULL,
    operator_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    action VARCHAR(40) NOT NULL,
    meta JSONB
);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_created_at
    ON ops_request_traces (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_request_id
    ON ops_request_traces (request_id);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_client_request_id
    ON ops_request_traces (client_request_id);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_upstream_request_id
    ON ops_request_traces (upstream_request_id);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_platform_time
    ON ops_request_traces (platform, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_protocol_time
    ON ops_request_traces (protocol_in, protocol_out, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_model_time
    ON ops_request_traces (requested_model, upstream_model, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_status_time
    ON ops_request_traces (status_code, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_request_traces_finish_reason_time
    ON ops_request_traces (finish_reason, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_ops_request_trace_audits_trace_time
    ON ops_request_trace_audits (trace_id, created_at DESC);

DO $$
BEGIN
    BEGIN
        CREATE EXTENSION IF NOT EXISTS pg_trgm;
    EXCEPTION
        WHEN OTHERS THEN
            RAISE NOTICE 'pg_trgm extension not created for ops_request_traces: %', SQLERRM;
    END;

    IF EXISTS (SELECT 1 FROM pg_extension WHERE extname = 'pg_trgm') THEN
        EXECUTE 'CREATE INDEX IF NOT EXISTS idx_ops_request_traces_search_text_trgm
                 ON ops_request_traces USING gin (search_text gin_trgm_ops)';
    ELSE
        RAISE NOTICE 'skip ops_request_traces trigram index because pg_trgm is unavailable';
    END IF;
END
$$;
