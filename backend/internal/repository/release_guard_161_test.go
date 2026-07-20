package repository

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestUpstream160To161CleanroomMatrixGuards(t *testing.T) {
	root := repositoryTestRepoRoot(t)

	matrix := readRepoFile(t, root, "docs", "upstream-sync", "upstream-v0.1.160-v0.1.161-cleanroom-sync-matrix.md")
	for _, expected := range []string{
		"`v0.1.160...v0.1.161`",
		"`Wei-Shaw/sub2api`",
		"`8bfbc5ca99bf2c0ac96e0f29ffd35eb6aca27e62`",
		"`19149ca196eeae4a4482e5299dc6fa4ba0b06c8c`",
		"`0.1.404`",
		"No `git pull`, `fetch`, merge, rebase, or cherry-pick",
		"Do not copy upstream LGPL/GPL",
		"MIT-only",
		"`/api-docs/*`",
		"`/admin/api-docs/*`",
		"`response.content_part.added`",
		"`response.output_text.done`",
		"`response.content_part.done`",
		"`response.completed.response.output`",
		"SSE frame boundary",
		"response.failed",
		"temp_unschedulable_rules",
		"model-scoped cooldown",
		"durable outbox",
		"invalid-credential rate limiting",
		"Authorization: Bearer",
		"Grok media",
		"`TARGETOS`",
		"`TARGETARCH`",
		"`TARGETVARIANT`",
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
	}
	require.Equal(t, expectedReleaseVersion, strings.TrimSpace(readRepoFile(t, root, "backend", "cmd", "server", "VERSION")))

	requireFileExists(t, root, "backend", "migrations", "164_api_key_auth_cache_invalidation_outbox.sql")
	for _, pattern := range []string{
		filepath.Join(root, "backend", "migrations", "183*.sql"),
		filepath.Join(root, "backend", "migrations", "184*.sql"),
	} {
		matches, err := filepath.Glob(pattern)
		require.NoError(t, err)
		require.Empty(t, matches, "must not adopt upstream migration set %s", pattern)
	}

	dockerfile := readRepoFile(t, root, "backend", "Dockerfile")
	for _, expected := range []string{"TARGETOS", "TARGETARCH", "TARGETVARIANT"} {
		require.Contains(t, dockerfile, expected)
	}

	assertNoAPIDocsRoutes(t, root)
}
