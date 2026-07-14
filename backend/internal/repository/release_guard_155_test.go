package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstream153To155CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.153-v0.1.155-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"Local baseline: `v0.1.387`",
		"`ec4a37da4f023fbaa4d46d2ee46a6e7f22e313d4`",
		"No `git pull`, `fetch`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`server.enable_server_timing=false`",
		"`http`, `redis`, and `db`",
		"`GET /admin/ops/system-logs` accepts `host`",
		"Grok OAuth SSO/device import and probing",
		"Reasoning `content:null`",
		"Responses Lite",
		"native Responses namespace verbatim including WSv2",
		"`gateway.openai_http2.send_ping_timeout_seconds`",
		"OpenAI images keepalive and final status",
		"Reset credit quota detection",
		"Long-context billing remains controlled",
		"`/v1/images/batches`",
	} {
		require.Contains(t, matrix, expected)
	}

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	defaults := readRepoFile(t, root, "backend", "internal", "config", "config_defaults.go")
	require.Contains(t, defaults, `server.enable_server_timing", false`)
	require.Contains(t, defaults, `gateway.openai_http2.send_ping_timeout_seconds", 30`)
	require.Contains(t, defaults, `gateway.openai_http2.ping_timeout_seconds", 15`)
	require.Contains(t, defaults, `grok.oauth.device_url`)

	serverTimingMiddleware := readRepoFile(t, root, "backend", "internal", "server", "middleware", "server_timing.go")
	require.Contains(t, serverTimingMiddleware, "servertiming.New")
	require.Contains(t, serverTimingMiddleware, "HeaderValue")

	sqlTiming := readRepoFile(t, root, "backend", "internal", "pkg", "servertiming", "sql.go")
	require.Contains(t, sqlTiming, "RegisterSQLDriver")
	require.Contains(t, sqlTiming, `Observe(ctx, "db")`)

	httpTiming := readRepoFile(t, root, "backend", "internal", "pkg", "servertiming", "http.go")
	require.Contains(t, httpTiming, `Observe(req.Context(), "http")`)

	redisTiming := readRepoFile(t, root, "backend", "internal", "pkg", "servertiming", "redis.go")
	require.Contains(t, redisTiming, `Observe(ctx, "redis")`)

	entInit := readRepoFile(t, root, "backend", "internal", "repository", "ent.go")
	require.Contains(t, entInit, "servertiming.RegisterSQLDriver")
	require.Contains(t, entInit, "entsql.OpenDB")

	opsHandler := readRepoFile(t, root, "backend", "internal", "handler", "admin", "ops_system_log_handler.go")
	require.Contains(t, opsHandler, `Host            string `)
	require.Contains(t, opsHandler, `strings.TrimSpace(c.Query("host"))`)

	opsWhere := readRepoFile(t, root, "backend", "internal", "repository", "ops_repo_system_logs_where.go")
	require.Contains(t, opsWhere, "strings.ToLower(strings.TrimSpace(filter.Host))")
	require.Contains(t, opsWhere, "l.extra->>'host'")

	grokOAuth := readRepoFile(t, root, "backend", "internal", "pkg", "grokoauth", "oauth.go")
	require.Contains(t, grokOAuth, "DefaultDeviceURL")
	require.Contains(t, grokOAuth, "DeviceGrantType")
	require.Contains(t, grokOAuth, "AuthorizationInputDeviceUserCode")

	grokDevice := readRepoFile(t, root, "backend", "internal", "service", "grok_oauth_device.go")
	require.Contains(t, grokDevice, "StartDeviceFlow")
	require.Contains(t, grokDevice, "PollDeviceToken")

	http2 := readRepoFile(t, root, "backend", "internal", "repository", "http_upstream_openai_http2.go")
	require.Contains(t, http2, "openAIHTTP2Keepalive")

	resetCredits := readRepoFile(t, root, "backend", "internal", "service", "openai_quota_service.go")
	require.Contains(t, resetCredits, "OpenAIResetCreditSummary")
	require.Contains(t, resetCredits, "sanitizeOpenAIResetCreditSummaries")

	frontendOps := readRepoFile(t, root, "frontend", "src", "views", "admin", "ops", "components", "OpsSystemLogTable.vue")
	require.Contains(t, frontendOps, "Host")
	require.Contains(t, frontendOps, "filters.host")

	for _, upstreamMigration := range []string{
		"174_add_usage_log_long_context_billing.sql",
		"175_add_ops_system_logs_host.sql",
		"175_default_openai_long_context_billing.sql",
		"175a_add_ops_system_logs_host_index_notx.sql",
		"176_channel_monitor_grok_provider.sql",
	} {
		_, err := os.Stat(filepath.Join(root, "backend", "migrations", upstreamMigration))
		require.True(t, os.IsNotExist(err), "must not adopt upstream migration %s", upstreamMigration)
	}

	assertNoAPIDocsRoutes(t, root)
}
