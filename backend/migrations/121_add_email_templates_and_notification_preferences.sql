CREATE TABLE IF NOT EXISTS email_templates (
    id BIGSERIAL PRIMARY KEY,
    template_key VARCHAR(64) NOT NULL,
    locale VARCHAR(16) NOT NULL DEFAULT 'en',
    subject TEXT NOT NULL,
    body TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    is_custom BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (template_key, locale)
);

CREATE INDEX IF NOT EXISTS idx_email_templates_key_locale
ON email_templates (template_key, locale);

CREATE TABLE IF NOT EXISTS user_notification_preferences (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    category VARCHAR(64) NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, category)
);

CREATE INDEX IF NOT EXISTS idx_user_notification_preferences_user
ON user_notification_preferences (user_id);

CREATE TABLE IF NOT EXISTS notification_dedupe_keys (
    dedupe_key TEXT PRIMARY KEY,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_notification_dedupe_keys_expires_at
ON notification_dedupe_keys (expires_at);
