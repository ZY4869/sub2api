//go:build integration

package repository

import (
	"context"
	"database/sql"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/migrations"
	"github.com/stretchr/testify/require"
)

func TestMigrationsRunner_IsIdempotent_AndSchemaIsUpToDate(t *testing.T) {
	tx := testTx(t)

	// Re-apply migrations to verify idempotency (no errors, no duplicate rows).
	require.NoError(t, ApplyMigrations(context.Background(), integrationDB))

	// schema_migrations should have at least the current migration set.
	var applied int
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT COUNT(*) FROM schema_migrations").Scan(&applied))
	require.GreaterOrEqual(t, applied, 7, "expected schema_migrations to contain applied migrations")

	// users: columns required by repository queries
	requireColumn(t, tx, "users", "username", "character varying", 100, false)
	requireColumn(t, tx, "users", "notes", "text", 0, false)
	requireColumn(t, tx, "users", "global_realtime_countdown_enabled", "boolean", 0, false)
	requireColumn(t, tx, "users", "account_realtime_countdown_enabled", "boolean", 0, false)
	requireColumn(t, tx, "users", "visual_preset_preference", "character varying", 32, false)
	requireColumn(t, tx, "users", "account_visual_preset_override", "character varying", 32, false)
	requireColumn(t, tx, "users", "account_today_stats_windows", "jsonb", 0, false)
	requireColumn(t, tx, "users", "account_today_stats_cycle_mode", "character varying", 32, false)
	requireColumn(t, tx, "users", "account_group_display_mode", "character varying", 32, false)
	requireColumn(t, tx, "users", "account_status_display_mode", "character varying", 32, false)
	requireColumnDefaultContains(t, tx, "users", "global_realtime_countdown_enabled", "false")
	requireColumnDefaultContains(t, tx, "users", "account_realtime_countdown_enabled", "true")
	requireColumnDefaultContains(t, tx, "users", "visual_preset_preference", "inherit")
	requireColumnDefaultContains(t, tx, "users", "account_visual_preset_override", "inherit")
	requireColumnDefaultContains(t, tx, "users", "account_today_stats_windows", "today")
	requireColumnDefaultContains(t, tx, "users", "account_today_stats_windows", "monthly")
	requireColumnDefaultContains(t, tx, "users", "account_today_stats_cycle_mode", "calendar")
	requireColumnDefaultContains(t, tx, "users", "account_group_display_mode", "full")
	requireColumnDefaultContains(t, tx, "users", "account_status_display_mode", "detailed")
	var globalRealtimeEnabled bool
	var accountRealtimeEnabled bool
	var visualPresetPreference string
	var accountVisualPresetOverride string
	var accountTodayStatsWindows string
	var accountTodayStatsCycleMode string
	var accountGroupDisplayMode string
	var accountStatusDisplayMode string
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2)
		 RETURNING global_realtime_countdown_enabled, account_realtime_countdown_enabled, visual_preset_preference, account_visual_preset_override, account_today_stats_windows::text, account_today_stats_cycle_mode, account_group_display_mode, account_status_display_mode`,
		"migration-realtime-defaults@example.com",
		"hash",
	).Scan(&globalRealtimeEnabled, &accountRealtimeEnabled, &visualPresetPreference, &accountVisualPresetOverride, &accountTodayStatsWindows, &accountTodayStatsCycleMode, &accountGroupDisplayMode, &accountStatusDisplayMode))
	require.False(t, globalRealtimeEnabled)
	require.True(t, accountRealtimeEnabled)
	require.Equal(t, "inherit", visualPresetPreference)
	require.Equal(t, "inherit", accountVisualPresetOverride)
	require.JSONEq(t, `["today","weekly","monthly","total"]`, accountTodayStatsWindows)
	require.Equal(t, "calendar", accountTodayStatsCycleMode)
	require.Equal(t, "full", accountGroupDisplayMode)
	require.Equal(t, "detailed", accountStatusDisplayMode)

	// accounts: schedulable and rate-limit fields
	requireColumn(t, tx, "accounts", "notes", "text", 0, true)
	requireColumn(t, tx, "accounts", "schedulable", "boolean", 0, false)
	requireColumn(t, tx, "accounts", "rate_limited_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "rate_limit_reset_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "overload_until", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "accounts", "session_window_status", "character varying", 20, true)
	requireColumn(t, tx, "accounts", "original_proxy_id", "bigint", 0, true)
	requireColumn(t, tx, "accounts", "original_proxy_name", "text", 0, true)
	requireIndex(t, tx, "accounts", "idx_accounts_original_proxy_id")

	// account_usage_periods: fixed-cycle account statistics period history.
	requireTable(t, tx, "account_usage_periods")
	requireColumn(t, tx, "account_usage_periods", "account_id", "bigint", 0, false)
	requireColumn(t, tx, "account_usage_periods", "window_type", "character varying", 32, false)
	requireColumn(t, tx, "account_usage_periods", "start_at", "timestamp with time zone", 0, false)
	requireColumn(t, tx, "account_usage_periods", "end_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "account_usage_periods", "reset_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "account_usage_periods", "source", "character varying", 32, false)
	requireColumn(t, tx, "account_usage_periods", "created_at", "timestamp with time zone", 0, false)
	requireColumn(t, tx, "account_usage_periods", "updated_at", "timestamp with time zone", 0, false)
	requireIndex(t, tx, "account_usage_periods", "idx_account_usage_periods_account_window_start")
	requireIndex(t, tx, "account_usage_periods", "idx_account_usage_periods_account_window_open")

	var usagePeriodBackfillChecksum sql.NullString
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		"SELECT checksum FROM schema_migrations WHERE filename = $1",
		"142_backfill_account_usage_periods.sql",
	).Scan(&usagePeriodBackfillChecksum))
	require.True(t, usagePeriodBackfillChecksum.Valid, "expected migration 142_backfill_account_usage_periods.sql to be recorded")

	// proxies: lifecycle metadata for expiry fallback recovery.
	requireColumn(t, tx, "proxies", "expires_at", "timestamp with time zone", 0, true)
	requireColumn(t, tx, "proxies", "expiry_remind_days", "integer", 0, false)
	requireColumn(t, tx, "proxies", "fallback_proxy_id", "bigint", 0, true)
	requireColumnDefaultContains(t, tx, "proxies", "expiry_remind_days", "0")
	requireIndex(t, tx, "proxies", "idx_proxies_expires_at")
	requireIndex(t, tx, "proxies", "idx_proxies_fallback_proxy_id")

	var proxyExpiryMigrationChecksum sql.NullString
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		"SELECT checksum FROM schema_migrations WHERE filename = $1",
		"138_proxy_expiry_fallback.sql",
	).Scan(&proxyExpiryMigrationChecksum))
	require.True(t, proxyExpiryMigrationChecksum.Valid, "expected migration 138_proxy_expiry_fallback.sql to be recorded")

	// api_keys: key length should be 128
	requireColumn(t, tx, "api_keys", "key", "character varying", 128, false)

	// redeem_codes: subscription fields
	requireColumn(t, tx, "redeem_codes", "group_id", "bigint", 0, true)
	requireColumn(t, tx, "redeem_codes", "validity_days", "integer", 0, false)
	requireColumn(t, tx, "redeem_codes", "expires_at", "timestamp with time zone", 0, true)
	requireIndex(t, tx, "redeem_codes", "idx_redeem_codes_expires_at")

	var redeemMigrationChecksum sql.NullString
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		"SELECT checksum FROM schema_migrations WHERE filename = $1",
		"119_add_redeem_code_expires_at.sql",
	).Scan(&redeemMigrationChecksum))
	require.True(t, redeemMigrationChecksum.Valid, "expected migration 119_add_redeem_code_expires_at.sql to be recorded")

	var legacyRedeemExpiresAt sql.NullTime
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		`INSERT INTO redeem_codes (code, type, value, status, notes, validity_days)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 RETURNING expires_at`,
		"MIGRATION-LEGACY-REDEEM",
		"balance",
		1,
		"unused",
		"",
		30,
	).Scan(&legacyRedeemExpiresAt))
	require.False(t, legacyRedeemExpiresAt.Valid, "legacy redeem codes without expires_at should stay non-expiring after migration")

	// auth_identities: DingTalk OAuth continues to rely on the shared social identity table introduced earlier.
	requireTable(t, tx, "auth_identities")
	requireColumn(t, tx, "auth_identities", "provider", "character varying", 32, false)
	requireColumn(t, tx, "auth_identities", "provider_user_id", "character varying", 255, false)
	requireColumn(t, tx, "auth_identities", "user_id", "bigint", 0, false)
	requireColumn(t, tx, "auth_identities", "email", "character varying", 255, false)
	requireColumn(t, tx, "auth_identities", "email_verified", "boolean", 0, false)
	requireColumn(t, tx, "auth_identities", "display_name", "character varying", 255, false)
	requireColumn(t, tx, "auth_identities", "avatar_url", "text", 0, false)
	requireIndex(t, tx, "auth_identities", "idx_auth_identities_provider_user")
	requireIndex(t, tx, "auth_identities", "idx_auth_identities_user_provider")

	var authIdentityUserID int64
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		`INSERT INTO users (email, password_hash) VALUES ($1, $2) RETURNING id`,
		"dingtalk-auth-identity@example.com",
		"hash",
	).Scan(&authIdentityUserID))

	var authIdentityID int64
	var authIdentityCreatedAt time.Time
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		`INSERT INTO auth_identities (
			provider, provider_user_id, user_id, email, email_verified, display_name, avatar_url
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, created_at`,
		"dingtalk",
		"ding-unionid-1",
		authIdentityUserID,
		"dingtalk-user@example.com",
		true,
		"DingTalk User",
		"https://example.com/avatar.png",
	).Scan(&authIdentityID, &authIdentityCreatedAt))
	require.NotZero(t, authIdentityID)
	require.False(t, authIdentityCreatedAt.IsZero())

	// usage_logs: billing_type used by filters/stats
	requireColumn(t, tx, "usage_logs", "billing_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "request_type", "smallint", 0, false)
	requireColumn(t, tx, "usage_logs", "openai_ws_mode", "boolean", 0, false)
	requireColumn(t, tx, "usage_logs", "status", "character varying", 16, false)
	requireColumn(t, tx, "usage_logs", "http_status", "integer", 0, true)
	requireColumn(t, tx, "usage_logs", "error_code", "character varying", 128, true)
	requireColumn(t, tx, "usage_logs", "error_message", "character varying", 1024, true)
	requireColumn(t, tx, "usage_logs", "simulated_client", "character varying", 32, true)

	// usage_billing_dedup: billing idempotency narrow table
	var usageBillingDedupRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup')").Scan(&usageBillingDedupRegclass))
	require.True(t, usageBillingDedupRegclass.Valid, "expected usage_billing_dedup table to exist")
	requireColumn(t, tx, "usage_billing_dedup", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_request_api_key")
	requireIndex(t, tx, "usage_billing_dedup", "idx_usage_billing_dedup_created_at_brin")

	var usageBillingDedupArchiveRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.usage_billing_dedup_archive')").Scan(&usageBillingDedupArchiveRegclass))
	require.True(t, usageBillingDedupArchiveRegclass.Valid, "expected usage_billing_dedup_archive table to exist")
	requireColumn(t, tx, "usage_billing_dedup_archive", "request_fingerprint", "character varying", 64, false)
	requireIndex(t, tx, "usage_billing_dedup_archive", "usage_billing_dedup_archive_pkey")

	// settings table should exist
	var settingsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.settings')").Scan(&settingsRegclass))
	require.True(t, settingsRegclass.Valid, "expected settings table to exist")

	// payment SQL repository schema (Airwallex clean-room payment path)
	requireTable(t, tx, "payment_orders")
	requireColumn(t, tx, "payment_orders", "order_no", "text", 0, false)
	requireColumn(t, tx, "payment_orders", "user_id", "bigint", 0, false)
	requireColumn(t, tx, "payment_orders", "product_type", "text", 0, false)
	requireColumn(t, tx, "payment_orders", "status", "text", 0, false)
	requireColumn(t, tx, "payment_orders", "provider", "text", 0, false)
	requireColumn(t, tx, "payment_orders", "provider_env", "text", 0, false)
	requireColumn(t, tx, "payment_orders", "amount_minor", "bigint", 0, false)
	requireColumn(t, tx, "payment_orders", "currency", "text", 0, false)
	requireColumn(t, tx, "payment_orders", "provider_intent_id", "text", 0, true)
	requireColumn(t, tx, "payment_orders", "resume_token_hash", "text", 0, false)
	requireColumn(t, tx, "payment_orders", "idempotency_key_hash", "text", 0, true)
	requireColumn(t, tx, "payment_orders", "snapshot_json", "jsonb", 0, false)
	requireIndex(t, tx, "payment_orders", "idx_payment_orders_provider_intent")
	requireIndex(t, tx, "payment_orders", "idx_payment_orders_user_idempotency")

	requireTable(t, tx, "payment_events")
	requireColumn(t, tx, "payment_events", "provider", "text", 0, false)
	requireColumn(t, tx, "payment_events", "provider_event_id", "text", 0, false)
	requireColumn(t, tx, "payment_events", "payload_hash", "text", 0, false)
	requireColumn(t, tx, "payment_events", "payload_redacted_json", "jsonb", 0, false)
	requireIndex(t, tx, "payment_events", "idx_payment_events_provider_event")

	requireTable(t, tx, "payment_refunds")
	requireColumn(t, tx, "payment_refunds", "refund_no", "text", 0, false)
	requireColumn(t, tx, "payment_refunds", "order_no", "text", 0, false)
	requireColumn(t, tx, "payment_refunds", "provider_refund_id", "text", 0, true)
	requireColumn(t, tx, "payment_refunds", "amount_minor", "bigint", 0, false)
	requireColumn(t, tx, "payment_refunds", "currency", "text", 0, false)
	requireColumn(t, tx, "payment_refunds", "idempotency_key_hash", "text", 0, true)
	requireIndex(t, tx, "payment_refunds", "idx_payment_refunds_order_idempotency")
	requireColumn(t, tx, "user_affiliate_ledger", "payment_order_id", "bigint", 0, true)
	requireIndex(t, tx, "user_affiliate_ledger", "idx_user_affiliate_ledger_payment_topup_dedup")

	// security_secrets table should exist
	var securitySecretsRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.security_secrets')").Scan(&securitySecretsRegclass))
	require.True(t, securitySecretsRegclass.Valid, "expected security_secrets table to exist")

	// user_allowed_groups table should exist
	var uagRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.user_allowed_groups')").Scan(&uagRegclass))
	require.True(t, uagRegclass.Valid, "expected user_allowed_groups table to exist")

	// user_subscriptions: deleted_at for soft delete support (migration 012)
	requireColumn(t, tx, "user_subscriptions", "deleted_at", "timestamp with time zone", 0, true)

	// orphan_allowed_groups_audit table should exist (migration 013)
	var orphanAuditRegclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.orphan_allowed_groups_audit')").Scan(&orphanAuditRegclass))
	require.True(t, orphanAuditRegclass.Valid, "expected orphan_allowed_groups_audit table to exist")

	// account_groups: created_at should be timestamptz
	requireColumn(t, tx, "account_groups", "created_at", "timestamp with time zone", 0, false)

	// user_allowed_groups: created_at should be timestamptz
	requireColumn(t, tx, "user_allowed_groups", "created_at", "timestamp with time zone", 0, false)

	// groups: OpenAI image protocol mode should exist with inherit default.
	requireColumn(t, tx, "groups", "image_protocol_mode", "character varying", 20, false)
	requireColumnDefaultContains(t, tx, "groups", "image_protocol_mode", "inherit")
	var imageProtocolMode string
	require.NoError(t, tx.QueryRowContext(
		context.Background(),
		"INSERT INTO groups (name) VALUES ($1) RETURNING image_protocol_mode",
		"migration-image-protocol-mode-default",
	).Scan(&imageProtocolMode))
	require.Equal(t, "inherit", imageProtocolMode)

	// ops_request_traces: request detail queries depend on Gemini/ billing metadata columns and indexes
	requireColumn(t, tx, "ops_request_traces", "gemini_surface", "character varying", 64, false)
	requireColumn(t, tx, "ops_request_traces", "billing_rule_id", "character varying", 128, false)
	requireColumn(t, tx, "ops_request_traces", "probe_action", "character varying", 64, false)
	requireIndex(t, tx, "ops_request_traces", "idx_ops_request_traces_gemini_surface_time")
	requireIndex(t, tx, "ops_request_traces", "idx_ops_request_traces_billing_rule_id")
	requireIndex(t, tx, "ops_request_traces", "idx_ops_request_traces_probe_action_time")

	// ops/deleted-key attribution added after local migration 133.
	requireTable(t, tx, "deleted_api_key_audits")
	requireColumn(t, tx, "deleted_api_key_audits", "api_key_id", "bigint", 0, false)
	requireColumn(t, tx, "deleted_api_key_audits", "user_id", "bigint", 0, false)
	requireColumn(t, tx, "deleted_api_key_audits", "name", "text", 0, false)
	requireColumn(t, tx, "deleted_api_key_audits", "key_prefix", "character varying", 32, false)
	requireColumn(t, tx, "deleted_api_key_audits", "deleted_at", "timestamp with time zone", 0, false)
	requireColumn(t, tx, "deleted_api_key_audits", "created_at", "timestamp with time zone", 0, false)
	requireIndex(t, tx, "deleted_api_key_audits", "idx_deleted_api_key_audits_user_deleted")
	requireIndex(t, tx, "deleted_api_key_audits", "idx_deleted_api_key_audits_key_prefix")

	requireColumn(t, tx, "ops_system_metrics", "ttft_sample_count", "bigint", 0, false)
	requireColumn(t, tx, "ops_metrics_hourly", "ttft_sample_count", "bigint", 0, false)
	requireColumn(t, tx, "ops_metrics_daily", "ttft_sample_count", "bigint", 0, false)
	requireColumn(t, tx, "ops_error_logs", "api_key_prefix", "character varying", 32, true)
	requireIndex(t, tx, "ops_error_logs", "idx_ops_error_logs_api_key_prefix_time")
	requireIndex(t, tx, "ops_error_logs", "idx_ops_error_logs_user_time")
	requireIndex(t, tx, "ops_error_logs", "idx_ops_error_logs_api_key_time")
}

func TestMigration142BackfillsLegacyAccountUsagePeriodsIdempotently(t *testing.T) {
	tx := testTx(t)
	ctx := context.Background()
	migrationBody, err := migrations.FS.ReadFile("142_backfill_account_usage_periods.sql")
	require.NoError(t, err)

	createdWithExpiry := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	expiresAt := createdWithExpiry.AddDate(0, 1, 0)
	createdWithoutExpiry := time.Date(2026, 2, 3, 4, 5, 6, 0, time.UTC)

	var expiryAccountID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO accounts (name, platform, type, status, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, "legacy-expiry", "anthropic", "oauth", "active", createdWithExpiry, expiresAt).Scan(&expiryAccountID))

	var fallbackAccountID int64
	require.NoError(t, tx.QueryRowContext(ctx, `
		INSERT INTO accounts (name, platform, type, status, created_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id
	`, "legacy-fallback", "anthropic", "oauth", "active", createdWithoutExpiry).Scan(&fallbackAccountID))

	require.NoError(t, execMigrationSQLTwice(ctx, tx, string(migrationBody)))

	var expiryStart time.Time
	var expiryEnd sql.NullTime
	var expiryReset sql.NullTime
	var expirySource string
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT start_at, end_at, reset_at, source
		FROM account_usage_periods
		WHERE account_id = $1 AND window_type = 'monthly'
	`, expiryAccountID).Scan(&expiryStart, &expiryEnd, &expiryReset, &expirySource))
	require.WithinDuration(t, createdWithExpiry, expiryStart, time.Second)
	require.True(t, expiryEnd.Valid)
	require.WithinDuration(t, expiresAt, expiryEnd.Time, time.Second)
	require.True(t, expiryReset.Valid)
	require.WithinDuration(t, expiresAt, expiryReset.Time, time.Second)
	require.Equal(t, "expiry", expirySource)

	var fallbackStart time.Time
	var fallbackEnd sql.NullTime
	var fallbackReset sql.NullTime
	var fallbackSource string
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT start_at, end_at, reset_at, source
		FROM account_usage_periods
		WHERE account_id = $1 AND window_type = 'monthly'
	`, fallbackAccountID).Scan(&fallbackStart, &fallbackEnd, &fallbackReset, &fallbackSource))
	require.WithinDuration(t, createdWithoutExpiry, fallbackStart, time.Second)
	require.False(t, fallbackEnd.Valid)
	require.False(t, fallbackReset.Valid)
	require.Equal(t, "fallback_30d", fallbackSource)

	var periodCount int
	require.NoError(t, tx.QueryRowContext(ctx, `
		SELECT COUNT(*)
		FROM account_usage_periods
		WHERE account_id IN ($1, $2) AND window_type = 'monthly'
	`, expiryAccountID, fallbackAccountID).Scan(&periodCount))
	require.Equal(t, 2, periodCount)
}

func execMigrationSQLTwice(ctx context.Context, tx *sql.Tx, body string) error {
	if _, err := tx.ExecContext(ctx, body); err != nil {
		return err
	}
	_, err := tx.ExecContext(ctx, body)
	return err
}

func requireIndex(t *testing.T, tx *sql.Tx, table, index string) {
	t.Helper()

	var exists bool
	err := tx.QueryRowContext(context.Background(), `
SELECT EXISTS (
	SELECT 1
	FROM pg_indexes
	WHERE schemaname = 'public'
	  AND tablename = $1
	  AND indexname = $2
)
`, table, index).Scan(&exists)
	require.NoError(t, err, "query pg_indexes for %s.%s", table, index)
	require.True(t, exists, "expected index %s on %s", index, table)
}

func requireTable(t *testing.T, tx *sql.Tx, table string) {
	t.Helper()

	var regclass sql.NullString
	require.NoError(t, tx.QueryRowContext(context.Background(), "SELECT to_regclass('public.' || $1)", table).Scan(&regclass))
	require.True(t, regclass.Valid, "expected %s table to exist", table)
}

func requireColumn(t *testing.T, tx *sql.Tx, table, column, dataType string, maxLen int, nullable bool) {
	t.Helper()

	var row struct {
		DataType string
		MaxLen   sql.NullInt64
		Nullable string
	}

	err := tx.QueryRowContext(context.Background(), `
SELECT
  data_type,
  character_maximum_length,
  is_nullable
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&row.DataType, &row.MaxLen, &row.Nullable)
	require.NoError(t, err, "query information_schema.columns for %s.%s", table, column)
	require.Equal(t, dataType, row.DataType, "data_type mismatch for %s.%s", table, column)

	if maxLen > 0 {
		require.True(t, row.MaxLen.Valid, "expected maxLen for %s.%s", table, column)
		require.Equal(t, int64(maxLen), row.MaxLen.Int64, "maxLen mismatch for %s.%s", table, column)
	}

	if nullable {
		require.Equal(t, "YES", row.Nullable, "nullable mismatch for %s.%s", table, column)
	} else {
		require.Equal(t, "NO", row.Nullable, "nullable mismatch for %s.%s", table, column)
	}
}

func requireColumnDefaultContains(t *testing.T, tx *sql.Tx, table, column, fragment string) {
	t.Helper()

	var columnDefault sql.NullString
	err := tx.QueryRowContext(context.Background(), `
SELECT column_default
FROM information_schema.columns
WHERE table_schema = 'public'
  AND table_name = $1
  AND column_name = $2
`, table, column).Scan(&columnDefault)
	require.NoError(t, err, "query default for %s.%s", table, column)
	require.True(t, columnDefault.Valid, "expected default for %s.%s", table, column)
	require.Contains(t, strings.ToLower(columnDefault.String), strings.ToLower(fragment), "default mismatch for %s.%s", table, column)
}
