package repository

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstream156CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.156-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"`v0.1.156^{}`",
		"`12f991dde8a58e183d4bd16a87ef6fd0df714757`",
		"local same-name tag",
		"No `git pull`, `fetch`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`POST /admin/accounts/:id/duplicate`",
		"`auth_mode=agentIdentity`",
		"Grok OAuth credential failover",
		"First-output timeout",
		"Responses Lite",
		"Anthropic direct chat bridge",
	} {
		require.Contains(t, matrix, expected)
	}

	license := readRepoFile(t, root, "LICENSE")
	require.True(t, strings.HasPrefix(license, "MIT License"), "root LICENSE must remain MIT")
	require.NotContains(t, license, "GNU LESSER GENERAL PUBLIC LICENSE")
	require.NotContains(t, license, "GNU GENERAL PUBLIC LICENSE")

	for _, name := range []string{"README.md", "README_CN.md", "README_EN.md"} {
		body := readRepoFile(t, root, name)
		require.Contains(t, body, "MIT License")
		require.NotContains(t, body, "LGPL-3.0")
	}
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	for _, forbidden := range []string{"COPYING", "SPONSORS.md", ".github/FUNDING.yml"} {
		_, err := os.Stat(filepath.Join(root, forbidden))
		require.True(t, os.IsNotExist(err), "must not adopt upstream release/governance artifact %s", forbidden)
	}

	assertNoAPIDocsRoutes(t, root)
}
