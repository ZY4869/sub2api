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

	require.Equal(t, "0.1.351", strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	var pkg struct {
		Version string `json:"version"`
		PNPM    struct {
			Overrides map[string]string `json:"overrides"`
		} `json:"pnpm"`
	}
	require.NoError(t, json.Unmarshal([]byte(readRepoFile(t, root, "frontend", "package.json")), &pkg))
	require.Equal(t, "0.1.351", pkg.Version)
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
