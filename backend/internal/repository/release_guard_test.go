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

	require.Equal(t, "0.1.373", strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	var pkg struct {
		Version string `json:"version"`
		PNPM    struct {
			Overrides map[string]string `json:"overrides"`
		} `json:"pnpm"`
	}
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, root, "frontend", "package.json")), &pkg))
	require.Equal(t, "0.1.373", pkg.Version)
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
