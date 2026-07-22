package repository

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstream161To162CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.161-v0.1.162-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"`v0.1.161...v0.1.162`",
		"`Wei-Shaw/sub2api`",
		"`19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`",
		"`27f094e0960ebd8e52de7ff7e763c6fec2ff4057`",
		"114 commits",
		"No `git pull`, `fetch`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL/CLA",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`GET /admin/settings/client-ip`",
		"`PUT /admin/settings/client-ip`",
		"`GET /admin/settings/image-batches/storage`",
		"`PUT /admin/settings/image-batches/storage`",
		"`POST /admin/settings/image-batches/storage/test`",
		"`prompt_cache_key`",
		"`insufficient_quota`",
		"`stop_reason:null`",
		"`ZY4869/sub2api`",
	} {
		require.Contains(t, matrix, expected)
	}

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")

	for _, name := range []string{"README.md", "README_CN.md", "README_EN.md"} {
		body := readRepoFile(t, root, name)
		require.Contains(t, body, "MIT License", "%s license section must stay MIT", name)
		require.NotContains(t, body, "LGPL-3.0", "%s must not be changed to LGPL", name)
		require.NotContains(t, body, "GNU LESSER GENERAL PUBLIC LICENSE", "%s must not embed LGPL text", name)
		require.NotContains(t, body, "GNU GENERAL PUBLIC LICENSE", "%s must not embed GPL text", name)
	}
	thirdParty := readRepoFile(t, root, "frontend", "THIRD_PARTY_LICENSES.md")
	require.NotContains(t, thirdParty, "LGPL")
	require.NotContains(t, thirdParty, "GPL")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	updateSvc := readRepoFile(t, root, "backend", "internal", "service", "update_service.go")
	require.Contains(t, updateSvc, `githubRepo     = "ZY4869/sub2api"`)
	require.NotContains(t, updateSvc, `githubRepo     = "Wei-Shaw/sub2api"`)
	githubClient := readRepoFile(t, root, "backend", "internal", "repository", "github_release_service.go")
	require.Contains(t, githubClient, `Authorization", "Bearer "`)
	require.NotRegexp(t, regexp.MustCompile(`slog\.[^\n]*githubToken|fmt\.(Errorf|Sprintf)\([^\n]*githubToken`), githubClient)

	requireFileExists(t, root, "backend", "internal", "service", "setting_service_client_ip.go")
	requireFileExists(t, root, "backend", "internal", "handler", "admin", "setting_handler_client_ip.go")
	requireFileExists(t, root, "backend", "internal", "service", "setting_service_image_batch_storage.go")
	requireFileExists(t, root, "backend", "internal", "handler", "admin", "setting_handler_image_batch_storage.go")
	requireFileExists(t, root, "backend", "internal", "service", "grok_gateway_tool_cache.go")
	requireFileExists(t, root, "frontend", "src", "components", "settings", "ClientIPSettingsCard.vue")
	requireFileExists(t, root, "frontend", "src", "components", "settings", "ImageBatchStorageSettingsCard.vue")

	for _, pattern := range []string{
		filepath.Join(root, "backend", "migrations", "185*.sql"),
		filepath.Join(root, "backend", "migrations", "186*.sql"),
		filepath.Join(root, "COPYING*"),
		filepath.Join(root, "CLA*"),
	} {
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)
		require.Empty(t, matches, "must not adopt upstream release/license artifact %s", pattern)
	}
	for _, path := range []string{
		filepath.Join(root, "logo.png"),
		filepath.Join(root, "logo.svg"),
		filepath.Join(root, "frontend", "src", "assets", "logo.png"),
		filepath.Join(root, "frontend", "src", "assets", "logo.svg"),
	} {
		_, err := os.Stat(path)
		require.True(t, os.IsNotExist(err), "must not copy upstream logo asset: %s", path)
	}

	assertNoAPIDocsRoutes(t, root)
}
