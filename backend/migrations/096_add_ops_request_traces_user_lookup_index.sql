CREATE INDEX IF NOT EXISTS idx_ops_request_traces_user_lookup
    ON ops_request_traces (user_id, api_key_id, request_id, created_at DESC)
    WHERE user_id IS NOT NULL
      AND api_key_id IS NOT NULL
      AND request_id IS NOT NULL;
