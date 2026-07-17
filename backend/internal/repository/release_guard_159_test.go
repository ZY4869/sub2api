package repository

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstream157To159CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.157-v0.1.159-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"`v0.1.156...v0.1.159`",
		"`Wei-Shaw/sub2api`",
		"`0.1.396`",
		"No `git pull`, `fetch`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`GET /v1/sub2api/billing`",
		"`prices_by_currency`",
		"`image_input_price`, `image_input_tokens`, and `image_input_cost`",
		"`audit_log_retention_days`",
		"`X-Sub2API-Step-Up-TOTP`",
		"`/v1/images/batches`",
		"`/images/tasks/:task_id`",
		"`submitted`, `running`, `completed`, `failed`, and `cancelled`",
		"Usage Windows",
		"resets left",
	} {
		require.Contains(t, matrix, expected)
	}

	auditService := readRepoFile(t, root, "backend", "internal", "service", "audit_log.go")
	require.Contains(t, auditService, "DefaultAuditLogRetentionDays")
	require.Contains(t, auditService, "AuditLogRetentionProvider")

	auditHandler := readRepoFile(t, root, "backend", "internal", "handler", "admin", "audit_log_handler.go")
	require.Contains(t, auditHandler, "admin.audit_logs.cleanup_expired")
	require.Contains(t, auditHandler, "RequireStepUpTotp")

	authBinding := readRepoFile(t, root, "backend", "internal", "service", "auth_session_binding.go")
	require.Contains(t, authBinding, "WithAuthSessionBinding")
	require.Contains(t, authBinding, "hashAuthSessionBindingValue")

	refreshTokenData := readRepoFile(t, root, "backend", "internal", "service", "refresh_token_cache.go")
	require.Contains(t, refreshTokenData, "client_ip_hash")
	require.Contains(t, refreshTokenData, "user_agent_hash")

	stepUpAPI := readRepoFile(t, root, "frontend", "src", "api", "admin", "stepUp.ts")
	require.Contains(t, stepUpAPI, "X-Sub2API-Step-Up-TOTP")

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")

	for _, name := range []string{"README.md", "README_CN.md", "README_EN.md"} {
		body := readRepoFile(t, root, name)
		require.Contains(t, body, "MIT License", "%s license section must stay MIT", name)
		require.NotContains(t, body, "LGPL-3.0", "%s must not be changed to LGPL", name)
	}
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
}
