CREATE TABLE IF NOT EXISTS tls_fingerprint_profiles (
    id BIGSERIAL PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT NULL,
    enable_grease BOOLEAN NOT NULL DEFAULT FALSE,
    cipher_suites JSONB NOT NULL DEFAULT '[]'::jsonb,
    curves JSONB NOT NULL DEFAULT '[]'::jsonb,
    point_formats JSONB NOT NULL DEFAULT '[]'::jsonb,
    signature_algorithms JSONB NOT NULL DEFAULT '[]'::jsonb,
    alpn_protocols JSONB NOT NULL DEFAULT '[]'::jsonb,
    supported_versions JSONB NOT NULL DEFAULT '[]'::jsonb,
    key_share_groups JSONB NOT NULL DEFAULT '[]'::jsonb,
    psk_modes JSONB NOT NULL DEFAULT '[]'::jsonb,
    extensions JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
