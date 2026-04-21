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
				"## Model Policy Rules",
				"`extra.model_scope_v2.entries[]` is the single source of truth",
				"must use `display_model_id` only",
			},
		},
		{
			path: filepath.Join(repoRoot, "backend", "internal", "service", "docs", "pages", "common.md"),
			mustContain: []string{
				"账号模型集合只来自两层：账号显式白名单 / 映射，或默认模型库",
				"本地策略投影和本地 availability snapshot",
			},
			mustNotContain: []string{
				"回退到保存的探测快照或实时探测结果",
			},
		},
		{
			path: filepath.Join(repoRoot, "backend", "internal", "service", "docs", "pages", "gemini.md"),
			mustContain: []string{
				"只暴露 display ID",
				"不能再当作公开模型 ID 查询详情",
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
