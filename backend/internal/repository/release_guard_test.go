package repository

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const expectedReleaseVersion = "0.1.408"

func TestSelectiveUpstreamAbsorptionReleaseGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")

	for _, name := range []string{"README.md", "README_CN.md", "README_EN.md"} {
		body := readRepoFile(t, root, name)
		require.Contains(t, body, "MIT License", "%s license section must stay MIT", name)
		require.NotContains(t, body, "LGPL-3.0", "%s must not be changed to LGPL", name)
	}

	thirdParty := readRepoFile(t, root, "frontend", "THIRD_PARTY_LICENSES.md")
	require.NotContains(t, thirdParty, "LGPL")
	require.NotContains(t, thirdParty, "GPL")

	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	var pkg struct {
		Version string `json:"version"`
		PNPM    struct {
			Overrides map[string]string `json:"overrides"`
		} `json:"pnpm"`
	}
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, root, "frontend", "package.json")), &pkg))
	require.Equal(t, expectedReleaseVersion, pkg.Version)
	require.Equal(t, "4.0.6", pkg.PNPM.Overrides["form-data"])

	assertNoAPIDocsRoutes(t, root)
}

func TestMigration146147Guards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	jitter := readRepoFile(t, root, "backend", "migrations", "146_channel_monitor_jitter.sql")
	require.Contains(t, jitter, "ADD COLUMN IF NOT EXISTS jitter_seconds")
	require.Contains(t, jitter, "channel_monitors_jitter_seconds_check")
	require.Contains(t, jitter, "interval_seconds - jitter_seconds >= 15")

	outbox := readRepoFile(t, root, "backend", "migrations", "147_scheduler_outbox_dedup_cleanup.sql")
	require.Contains(t, outbox, "ADD COLUMN IF NOT EXISTS dedup_key")
	require.Contains(t, outbox, "CREATE UNIQUE INDEX IF NOT EXISTS idx_scheduler_outbox_dedup_key")
	require.Contains(t, outbox, "WHERE dedup_key IS NOT NULL")
	require.Contains(t, outbox, "CREATE INDEX IF NOT EXISTS idx_scheduler_outbox_id_created_at")
}

func TestUpstream137To139CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "上游v0.1.137-v0.1.139_洁净重写同步矩阵_20260628.md")
	for _, expected := range []string{
		"eba9bea959dad0c6db30994870c60085965e2fd5",
		"6936687870e9f5f21ec814ec8c7fcec9d7b37c10",
		"9a0fbcc87dc14c3ad4f87c3fad951a320f109050",
		"本地基线：`0.1.368`",
		"不执行 `git pull`、merge、rebase、cherry-pick",
		"不复制 upstream LGPL",
		"SUB2API_JWT",
		"jitter_seconds",
		"content_moderation_cyber_policy_enabled",
		"claude_oauth_system_prompt_blocks",
		"affiliate_rebate",
		"refresh_token_invalidated",
		"app_session_terminated",
		"CLAUDE_CODE_ATTRIBUTION_HEADER",
		"form-data",
		"SELinux",
		"model_not_found",
	} {
		require.Contains(t, matrix, expected)
	}

	cli := readRepoFile(t, root, "skills", "sub2api-admin", "scripts", "sub2api-admin.js")
	require.Contains(t, cli, "process.env.SUB2API_ADMIN_TOKEN || process.env.SUB2API_JWT")

	keyModal := readRepoFile(t, root, "frontend", "src", "components", "keys", "UseKeyModal.vue")
	require.Contains(t, keyModal, "CLAUDE_CODE_ATTRIBUTION_HEADER: '0'")

	envExample := readRepoFile(t, root, "deploy", ".env.example")
	require.Contains(t, envExample, "SELINUX_VOLUME_LABEL")
	require.Contains(t, envExample, ":Z")
}

func TestUpstream144CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.144-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"a10dc955189e9c7d70dcbb0f0a6334b2932bbde0",
		"MIT-only",
		"0.1.375",
		"v0.1.145+",
		"codex-sessions/import",
		"seven_day_fable",
		"codex_image_tool_policy",
		"UsageRecordOverflowPolicySync",
		"target_model_id",
		"no `/api-docs/*`",
	} {
		require.Contains(t, matrix, expected)
	}

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
}

func TestUpstream145To146CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.145-v0.1.146-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"b023f1a507039c620c12a07f22e8d7e2a0f70df5",
		"3aee00f59fc2cb6c46e9ee7631b9d799a736b596",
		"MIT-only",
		"0.1.376",
		"do not run `git pull`, merge, rebase, cherry-pick",
		"copy upstream LGPL",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`current_concurrency`",
		"`/responses/compact`",
		"openai_advanced_scheduler",
		"`request_headers`",
		"payment_subscription_usd_to_cny_rate",
		"`gpt-5.6-sol`",
		"Antigravity OAuth 401",
		"Model lists and runtime support checks continue to use local policy projection",
	} {
		require.Contains(t, matrix, expected)
	}

	coreDefaults := readRepoFile(t, root, "backend", "internal", "config", "config_defaults.go")
	require.Contains(t, coreDefaults, `security.url_allowlist.allow_insecure_http", false`)
	require.Contains(t, coreDefaults, `security.url_allowlist.allow_private_hosts", false`)

	prodCompose := readRepoFile(t, root, "deploy", "docker-compose.yml")
	require.Contains(t, prodCompose, "SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=${SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP:-false}")

	localCompose := readRepoFile(t, root, "deploy", "docker-compose.local.yml")
	require.Contains(t, localCompose, "SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=${SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP:-true}")
	require.Contains(t, localCompose, "For production, set")

	devCompose := readRepoFile(t, root, "deploy", "docker-compose.dev.yml")
	require.Contains(t, devCompose, "Local source builds allow HTTP upstream URLs for dev/test only.")
	require.Contains(t, devCompose, "SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=${SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP:-true}")

	envExample := readRepoFile(t, root, "deploy", ".env.example")
	require.Contains(t, envExample, "Set SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=false")
	require.Contains(t, envExample, "SECURITY_URL_ALLOWLIST_ALLOW_INSECURE_HTTP=true")

	dtoSettings := readRepoFile(t, root, "backend", "internal", "handler", "dto", "settings.go")
	require.Contains(t, dtoSettings, `json:"payment_subscription_usd_to_cny_rate"`)
	require.Contains(t, dtoSettings, `json:"openai_advanced_scheduler_enabled"`)

	apiKeyDTO := readRepoFile(t, root, "backend", "internal", "handler", "dto", "types.go")
	require.Contains(t, apiKeyDTO, `json:"current_concurrency"`)

	headerOverride := readRepoFile(t, root, "backend", "internal", "service", "account_header_override.go")
	for _, blocked := range []string{
		`"authorization"`,
		`"x-api-key"`,
		`"cookie"`,
		`"host"`,
	} {
		require.Contains(t, headerOverride, blocked)
	}

	endpoint := readRepoFile(t, root, "backend", "internal", "handler", "endpoint.go")
	require.Contains(t, endpoint, "/v1/responses/compact")

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
}

func TestUpstream146To150CleanroomMatrixAndRollbackGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.146-v0.1.150-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"d7a6a4513a58b082922dfb8bd80f36cbe6b8a4c4",
		"ba1f130d283daae4de70322cc17f01673808e986",
		"dd1a116f4879992f04bb6ddbc77069bd503c3f4a",
		"0dec1ad2922ff8c9d27b67f8a31dfb35bce1902b",
		"Local baseline: `v0.1.379`",
		"No `git pull`, `fetch`, merge, rebase, or cherry-pick",
		"MIT-only",
		"`check-updates` exposes up to three older trusted local release candidates",
		"`rollback.target_version`",
		"`Wei-Shaw/sub2api`",
		"Batch image generation",
		"`/v1/images/batches`",
		"`display_model_id`",
		"`target_model_id` remains internal",
	} {
		require.Contains(t, matrix, expected)
	}

	policy := readRepoFile(t, root, "backend", "internal", "service", "update_release_policy.go")
	require.Contains(t, policy, `allowedDownloadHost = "github.com"`)
	require.Contains(t, policy, `githubRepo+"/releases/download/"`)
	require.NotContains(t, policy, "objects.githubusercontent.com")
	require.NotContains(t, policy, "Wei-Shaw/sub2api/releases/download")
	require.NotContains(t, policy, "hub.docker.com")

	updateSvc := readRepoFile(t, root, "backend", "internal", "service", "update_service.go")
	require.Contains(t, updateSvc, `githubRepo     = "ZY4869/sub2api"`)
	require.Contains(t, updateSvc, "RollbackToVersion")
	require.Contains(t, updateSvc, "SYSTEM_ROLLBACK_SIGNATURE_REQUIRED")
	require.Contains(t, updateSvc, "SYSTEM_ROLLBACK_SIGNATURE_VERIFIER_UNCONFIGURED")

	imageBatchConfig := readRepoFile(t, root, "backend", "internal", "config", "config_types_core.go")
	require.Contains(t, imageBatchConfig, "SettlementRetryLimit")
	require.Contains(t, imageBatchConfig, "OutputCleanupAfterHours")
	require.Contains(t, imageBatchConfig, "DownloadConcurrency")

	imageBatchRepo := readRepoFile(t, root, "backend", "internal", "service", "image_batch_repository.go")
	require.Contains(t, imageBatchRepo, "IncrementJobSettlementRetry")
	require.Contains(t, imageBatchRepo, "CleanupExpiredOutputs")

	opsRoutes := readRepoFile(t, root, "backend", "internal", "server", "routes", "admin.go")
	require.Contains(t, opsRoutes, `runtime.GET("/image-batch"`)

	imageBatchMetrics := readRepoFile(t, root, "backend", "internal", "service", "image_batch_metrics.go")
	require.Contains(t, imageBatchMetrics, "OutputsCleanedJobs")

	frontendSystemAPI := readRepoFile(t, root, "frontend", "src", "api", "admin", "system.ts")
	require.Contains(t, frontendSystemAPI, "RollbackVersionInfo")
	require.Contains(t, frontendSystemAPI, "rollback_versions")
	require.Contains(t, frontendSystemAPI, "target_version")

	versionBadge := readRepoFile(t, root, "frontend", "src", "components", "common", "VersionBadge.vue")
	require.Contains(t, versionBadge, "rollbackVersions")
	require.Contains(t, versionBadge, "version.rollbackFailed")
	require.Contains(t, versionBadge, "metadata?.error_id")

	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
}

func TestUpstream150To151CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.150-v0.1.151-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"Local baseline: `v0.1.379`",
		"No `git pull`, `fetch`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`157_allow_cyber_blocked_usage_request_type.sql`",
		"do not add upstream migration `173`",
		"`openai_fast_policy_settings.rules[].user_ids`",
		"OpenAIRealSSEStarted",
		"`response.failed`",
		"`gpt-5.6-{sol,terra,luna}`",
		"`max`",
		"`reasoning` / `reasoning_effort`",
		"`image_gen`",
		"setup-token",
		"`/v1/models` and `/v1/chat/completions` without key return 401",
	} {
		require.Contains(t, matrix, expected)
	}

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")

	migration157 := filepath.Join(root, "backend", "migrations", "157_allow_cyber_blocked_usage_request_type.sql")
	_, err := os.Stat(migration157)
	require.NoError(t, err)
	migration173 := filepath.Join(root, "backend", "migrations", "173_allow_cyber_blocked_usage_request_type.sql")
	_, err = os.Stat(migration173)
	require.True(t, os.IsNotExist(err), "must not adopt upstream migration 173")

	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
}

func TestUpstream151To152CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.151-v0.1.152-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"Local baseline: `v0.1.386`",
		"`553ab6f911247963eb368fcf6ac1dcb65d5495b1`",
		"No `git pull`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`158_add_group_web_search_price_per_call.sql`",
		"do not adopt upstream migration numbering",
		"`alpha/search`",
		"`web_search_price_per_call`",
		"`0.01` USD per call",
		"`custom`, `tool_search`, namespace tools",
		"`cache_creation_input_tokens`",
		"Grok xAI API key and OAuth routing",
		"OpenAIFastPolicyUserSelector",
		"Ops capture writer release safety",
	} {
		require.Contains(t, matrix, expected)
	}

	migration158 := readRepoFile(t, root, "backend", "migrations", "158_add_group_web_search_price_per_call.sql")
	require.Contains(t, migration158, "web_search_price_per_call")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
}

func TestUpstream152To153CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.152-v0.1.153-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"Local baseline: `v0.1.386`",
		"`a2bc133`",
		"No `git pull`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`159_add_usage_logs_api_key_recent_ip_index_notx.sql`",
		"do not adopt upstream migration `174`",
		"ZY4869/sub2api",
		"ghcr.io/zy4869/sub2api",
		"`/v1/videos/edits`",
		"`/v1/videos/extensions`",
		"Grok API Key third-party base URL",
		"OpenAI OAuth `plan_type` manual override",
		"Deprecated payment API removal",
		"DataTable small data jitter",
	} {
		require.Contains(t, matrix, expected)
	}

	appleScript := readRepoFile(t, root, "deploy", "apple-container.sh")
	require.Contains(t, appleScript, `DEFAULT_GITHUB_REPO="ZY4869/sub2api"`)
	require.Contains(t, appleScript, `DEFAULT_SUB2API_IMAGE="ghcr.io/zy4869/sub2api:latest"`)
	require.NotContains(t, appleScript, "Wei-Shaw/sub2api")
	require.NotContains(t, appleScript, "ghcr.io/wei-shaw")
	require.NotContains(t, appleScript, "LGPL")

	appleDoc := readRepoFile(t, root, "deploy", "APPLE_CONTAINER.md")
	require.Contains(t, appleDoc, "`ghcr.io/zy4869/sub2api:latest`")
	require.Contains(t, appleDoc, "`ZY4869/sub2api`")
	require.NotContains(t, appleDoc, "LGPL")

	migration159 := readRepoFile(t, root, "backend", "migrations", "159_add_usage_logs_api_key_recent_ip_index_notx.sql")
	require.Contains(t, migration159, "CREATE INDEX CONCURRENTLY IF NOT EXISTS idx_usage_logs_api_key_created_ip_not_null")
	require.Contains(t, migration159, "ON usage_logs (api_key_id, created_at DESC, ip_address)")
	require.Contains(t, migration159, "WHERE api_key_id IS NOT NULL AND ip_address IS NOT NULL")
	_, err := os.Stat(filepath.Join(root, "backend", "migrations", "174_add_usage_logs_api_key_recent_ip_index_notx.sql"))
	require.True(t, os.IsNotExist(err), "must not adopt upstream migration numbering")

	userRoutes := readRepoFile(t, root, "backend", "internal", "server", "routes", "user.go")
	require.Contains(t, userRoutes, `payments.POST("/orders", h.Payment.CreateOrder)`)
	require.Contains(t, userRoutes, `payments.GET("/orders/:order_no/resume", h.Payment.ResumeOrderByOrderNo)`)
	require.Contains(t, userRoutes, `payments.GET("/resume/:resume_token", h.Payment.ResumeOrder)`)
	require.Contains(t, userRoutes, `payments.POST("/orders/:order_no/cancel", h.Payment.CancelOrder)`)
	require.NotContains(t, userRoutes, "channel_config")
	require.NotContains(t, userRoutes, "provider_config")

	adminRoutes := readRepoFile(t, root, "backend", "internal", "server", "routes", "admin.go")
	require.Contains(t, adminRoutes, `payments.POST("/orders/:order_no/refund", h.Admin.Payment.RefundOrder)`)

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
}

func TestUpstream163To165CleanroomMatrixAndLicenseGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.163-v0.1.165-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"`v0.1.162=27f094e0960ebd8e52de7ff7e763c6fec2ff4057`",
		"`v0.1.163=d0bdd7e771636a8d315f542cafd39484f39bd60c`",
		"`v0.1.164=cd8bb98c44303b2c8f04c0da340447c992f0cb7d`",
		"`v0.1.165=e9a58c1cb8b5ef626a75c93b4d953fde5e67aa29`",
		"本地基线：`0.1.408`",
		"clean-room 本地重写",
		"不执行 `git pull`、`git fetch`、merge、rebase、cherry-pick",
		"MIT-only",
		"`platform=composite`",
		"`display_model_id`",
		"`target_model_id` 仅内部转发",
		"`groups.allow_live` 默认关闭",
		"`/v1/live`",
		"`/backend-api/codex/realtime/calls`",
		"`max_reasoning_effort`",
		"`reasoning_effort_mappings`",
		"`usage_logs.session_id`",
		"`users.email_alias`",
		"Ollama Cloud 用量请求驱动刷新与 15 分钟下限 | 不适用",
		"Alipay 移动端 deep link | 明确排除",
		"`claude-opus-5` 模型/定价/Bedrock 映射/前端预设/限流 scope | 已覆盖",
		"`model_registry_available_models_bootstrap_v20260726`",
		"图像请求日志记录 `quality`/`size` | 已覆盖",
		"`169_add_usage_log_image_quality.sql`",
		"`usage_logs.image_quality`",
		"只做日志诊断，不改变公开计费语义",
		"Airwallex",
		"`payment_mobile_force_qrcode_enabled`",
		"前端后台分组 UI 只把 `composite` 暴露给分组管理",
	} {
		require.Contains(t, matrix, expected)
	}
	for _, forbidden := range []string{
		"Ollama Cloud 用量请求驱动刷新与 15 分钟下限 | 已覆盖",
		"Alipay 移动端 deep link | 已覆盖",
		"`claude-opus-5` 模型/定价/Bedrock 映射/前端预设/限流 scope | 明确排除",
		"图像请求日志记录 `quality`/`size` | 部分覆盖",
		"`quality` 字段，本轮不新增数据语义",
		"Ollama PG<=16 due 判定、会话脱敏、刷新饥饿 | 已覆盖",
	} {
		require.NotContains(t, matrix, forbidden)
	}

	for _, migration := range []string{
		"165_add_group_live_reasoning_fields.sql",
		"166_create_composite_model_routes.sql",
		"167_add_usage_log_session_id.sql",
		"168_add_user_email_alias.sql",
		"169_add_usage_log_image_quality.sql",
	} {
		_, err := os.Stat(filepath.Join(root, "backend", "migrations", migration))
		require.NoError(t, err)
	}

	groupSchema := readRepoFile(t, root, "backend", "ent", "schema", "group.go")
	require.Contains(t, groupSchema, "allow_live")
	require.Contains(t, groupSchema, "max_reasoning_effort")
	require.Contains(t, groupSchema, "reasoning_effort_mappings")

	compositeSchema := readRepoFile(t, root, "backend", "ent", "schema", "composite_model_route.go")
	require.Contains(t, compositeSchema, "display_model_id")
	require.Contains(t, compositeSchema, "target_group_id")
	require.Contains(t, compositeSchema, "target_model_id")

	userSchema := readRepoFile(t, root, "backend", "ent", "schema", "user.go")
	require.Contains(t, userSchema, "email_alias")
	userAliasMigration := readRepoFile(t, root, "backend", "migrations", "168_add_user_email_alias.sql")
	require.Contains(t, userAliasMigration, "users_email_alias_unique_active")

	usageLogSchema := readRepoFile(t, root, "backend", "ent", "schema", "usage_log.go")
	require.Contains(t, usageLogSchema, "image_quality")
	usageLogDTO := readRepoFile(t, root, "backend", "internal", "handler", "dto", "types.go")
	require.Contains(t, usageLogDTO, `json:"image_quality"`)
	adminUsageTable := readRepoFile(t, root, "frontend", "src", "components", "admin", "usage", "UsageTable.vue")
	require.Contains(t, adminUsageTable, "row.image_quality")
	userUsageTable := readRepoFile(t, root, "frontend", "src", "views", "user", "usage", "UsageTable.vue")
	require.Contains(t, userUsageTable, "row.image_quality")

	modelRegistrySeed := readRepoFile(t, root, "backend", "internal", "modelregistry", "registry_seed.json")
	require.Contains(t, modelRegistrySeed, `"id": "claude-opus-5"`)
	modelCatalogSeed := readRepoFile(t, root, "backend", "internal", "service", "model_catalog_seed.json")
	require.Contains(t, modelCatalogSeed, `"model":"claude-opus-5"`)
	pricingCatalog := readRepoFile(t, root, "backend", "resources", "model-pricing", "model_prices_and_context_window.json")
	require.Contains(t, pricingCatalog, `"claude-opus-5"`)
	bedrockMapping := readRepoFile(t, root, "backend", "internal", "domain", "constants.go")
	require.Contains(t, bedrockMapping, `"claude-opus-5":`)
	require.Contains(t, bedrockMapping, `"us.anthropic.claude-opus-5"`)
	modelAvailability := readRepoFile(t, root, "backend", "internal", "service", "model_registry_availability.go")
	require.Contains(t, modelAvailability, `"claude-opus-5"`)
	require.Contains(t, modelAvailability, "ensureAvailableModelsBootstrapV20260726")
	frontendWhitelist := readRepoFile(t, root, "frontend", "src", "composables", "useModelWhitelist.ts")
	require.Contains(t, frontendWhitelist, `"claude-opus-5"`)
	generatedRegistry := readRepoFile(t, root, "frontend", "src", "generated", "modelRegistry.ts")
	require.Contains(t, generatedRegistry, `"id": "claude-opus-5"`)

	adminRoutes := readRepoFile(t, root, "backend", "internal", "server", "routes", "admin.go")
	require.Contains(t, adminRoutes, `groups.GET("/:id/composite-routes"`)
	require.Contains(t, adminRoutes, `groups.PUT("/:id/composite-routes"`)
	require.Contains(t, adminRoutes, `groups.POST("/:id/composite-routes/preview"`)

	gatewayRoutes := readRepoFile(t, root, "backend", "internal", "server", "routes", "gateway.go")
	require.Contains(t, gatewayRoutes, `gateway.GET("/live"`)
	require.Contains(t, gatewayRoutes, `codexBackend.GET("/realtime/calls"`)
	require.Contains(t, gatewayRoutes, `codexBackend.POST("/realtime/calls"`)

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))
	assertNoAPIDocsRoutes(t, root)
	assertNoCopyleftLicenseTextInRuntimeCode(t, root)
}

func repositoryTestRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

func readRepoFile(t *testing.T, root string, parts ...string) string {
	t.Helper()
	body, err := os.ReadFile(filepath.Join(append([]string{root}, parts...)...))
	require.NoError(t, err)
	return string(body)
}

func assertNoAPIDocsRoutes(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{
		filepath.Join(root, "backend", "internal", "handler"),
		filepath.Join(root, "backend", "internal", "server"),
		filepath.Join(root, "frontend", "src", "router"),
	} {
		require.NoError(t, filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			require.NoError(t, err)
			if entry.IsDir() {
				if entry.Name() == "__tests__" {
					return filepath.SkipDir
				}
				return nil
			}
			base := entry.Name()
			if strings.Contains(base, "_test.") || strings.Contains(base, ".spec.") {
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".ts", ".vue", ".js":
			default:
				return nil
			}
			body, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			text := string(body)
			require.NotContains(t, text, `"/api-docs`)
			require.NotContains(t, text, `'/api-docs`)
			require.NotContains(t, text, "`/api-docs")
			require.NotContains(t, text, `"/admin/api-docs`)
			require.NotContains(t, text, `'/admin/api-docs`)
			require.NotContains(t, text, "`/admin/api-docs")
			return nil
		}))
	}
}

func assertNoCopyleftLicenseTextInRuntimeCode(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{
		filepath.Join(root, "backend", "cmd"),
		filepath.Join(root, "backend", "ent", "schema"),
		filepath.Join(root, "backend", "internal"),
		filepath.Join(root, "frontend", "src"),
	} {
		require.NoError(t, filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			require.NoError(t, err)
			if entry.IsDir() {
				if entry.Name() == "__tests__" || entry.Name() == "__generated__" {
					return filepath.SkipDir
				}
				return nil
			}
			if strings.Contains(entry.Name(), "_test.") || strings.Contains(entry.Name(), ".spec.") {
				return nil
			}
			switch filepath.Ext(path) {
			case ".go", ".ts", ".vue", ".js":
			default:
				return nil
			}
			body, readErr := os.ReadFile(path)
			require.NoError(t, readErr)
			text := string(body)
			require.NotContains(t, text, "GNU LESSER GENERAL PUBLIC LICENSE")
			require.NotContains(t, text, "GNU GENERAL PUBLIC LICENSE")
			require.NotContains(t, text, "LGPL-3.0")
			require.NotContains(t, text, "Contributor License Agreement")
			return nil
		}))
	}
}
