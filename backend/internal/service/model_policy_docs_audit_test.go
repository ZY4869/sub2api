package service

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelPolicyDocsAndGuideStayAligned(t *testing.T) {
	t.Parallel()

	_, currentFile, _, ok := runtime.Caller(0)
	require.True(t, ok)

	repoRoot := filepath.Clean(filepath.Join(filepath.Dir(currentFile), "..", "..", ".."))
	auditFiles := []struct {
		path           string
		mustContain    []string
		mustNotContain []string
	}{
		{
			path: filepath.Join(repoRoot, "AGENTS.md"),
			mustContain: []string{
				"## Model Example Template Sync Rules",
				"model_catalog_public_example_templates.go",
				"Do not add `/api-docs/*` or `/admin/api-docs/*` routes",
				"## Model Policy Rules",
				"`extra.model_scope_v2.entries[]` is the single source of truth",
				"must use `display_model_id` only",
			},
		},
		{
			path: filepath.Join(repoRoot, "backend", "internal", "service", "model_catalog_public_example_templates.go"),
			mustContain: []string{
				"publicModelCatalogExamplePages",
				"Authorization: Bearer sk-你的站内Key",
				"/v1/responses",
				"/v1/chat/completions",
				"/v1/messages",
				":generateContent",
			},
			mustNotContain: []string{
				"api-docs",
				"admin/api-docs",
			},
		},
		{
			path: filepath.Join(repoRoot, "backend", "internal", "service", "model_catalog_public_examples.go"),
			mustContain: []string{
				"selectPublicModelCatalogExampleSpec",
				"publicModelCatalogExampleTemplateMarkdown",
			},
		},
		{
			path: filepath.Join(repoRoot, "backend", "internal", "service", "docs", "model-policy-terms.md"),
			mustContain: []string{
				"# Model Policy Terms",
				"## Display Model",
				"## Target Model",
				"## Default Library",
				"## Policy Projection",
				"## Availability Snapshot",
			},
		},
	}

	for _, auditFile := range auditFiles {
		content, err := os.ReadFile(auditFile.path)
		require.NoError(t, err, auditFile.path)
		text := string(content)
		for _, expected := range auditFile.mustContain {
			require.Contains(t, text, expected, auditFile.path)
		}
		for _, forbidden := range auditFile.mustNotContain {
			require.NotContains(t, text, forbidden, auditFile.path)
		}
	}
}
