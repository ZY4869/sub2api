package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstream159To160PromptAuditCleanroomGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	requireFileExists(t, root, "backend", "migrations", "162_prompt_audit.sql")
	requireFileExists(t, root, "backend", "migrations", "163_prompt_audit_full_prompt.sql")
	requireFileMissing(t, root, "backend", "migrations", "181_prompt_audit.sql")
	requireFileMissing(t, root, "backend", "migrations", "182_prompt_audit_full_prompt.sql")
	requireFileMissing(t, root, "openspec", "changes", "add-openai-compatible-prompt-audit", "source-freeze")

	assertNoAPIDocsRoutes(t, root)
	assertNoCopiedUpstreamPromptAuditArtifacts(t, root)
}

func assertNoCopiedUpstreamPromptAuditArtifacts(t *testing.T, root string) {
	t.Helper()
	for _, dir := range []string{
		filepath.Join(root, "frontend", "src"),
		filepath.Join(root, "backend", "internal"),
		filepath.Join(root, "docs"),
	} {
		require.NoError(t, filepath.WalkDir(dir, func(path string, entry os.DirEntry, err error) error {
			require.NoError(t, err)
			if entry.IsDir() {
				if entry.Name() == "source-freeze" {
					t.Fatalf("must not copy upstream source-freeze directory: %s", path)
				}
				return nil
			}
			name := strings.ToLower(entry.Name())
			require.False(t, strings.Contains(name, "sponsor"), "must not copy sponsor material: %s", path)
			require.False(t, strings.Contains(name, "lgpl"), "must not copy LGPL text: %s", path)
			return nil
		}))
	}
}

func requireFileExists(t *testing.T, root string, parts ...string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(append([]string{root}, parts...)...))
	require.NoError(t, err)
}

func requireFileMissing(t *testing.T, root string, parts ...string) {
	t.Helper()
	_, err := os.Stat(filepath.Join(append([]string{root}, parts...)...))
	require.True(t, os.IsNotExist(err), "path must not exist: %s", filepath.Join(parts...))
}
