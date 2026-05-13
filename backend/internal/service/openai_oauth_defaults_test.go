package service

import "testing"

import "github.com/stretchr/testify/require"

func TestResolveOpenAIOAuthDefaultAllowedModels(t *testing.T) {
	t.Run("free excludes image model", func(t *testing.T) {
		require.Equal(t, []string{
			"gpt-5.2",
			"gpt-5.4",
			"gpt-5.4-mini",
			"gpt-5.5",
		}, ResolveOpenAIOAuthDefaultAllowedModels("free", 0))
	})

	t.Run("paid keeps current defaults", func(t *testing.T) {
		require.Equal(t, []string{
			"gpt-image-2",
			"gpt-5.2",
			"gpt-5.4",
			"gpt-5.4-mini",
			"gpt-5.5",
		}, ResolveOpenAIOAuthDefaultAllowedModels("plus", 0))
	})

	t.Run("pro appends spark", func(t *testing.T) {
		require.Equal(t, []string{
			"gpt-image-2",
			"gpt-5.2",
			"gpt-5.4",
			"gpt-5.4-mini",
			"gpt-5.5",
			"gpt-5.3-codex-spark",
		}, ResolveOpenAIOAuthDefaultAllowedModels("pro", 20))
	})
}

func TestBuildOpenAIOAuthDefaultModelScopeExtra(t *testing.T) {
	extra := BuildOpenAIOAuthDefaultModelScopeExtra(map[string]any{
		"privacy_mode": "disabled",
	}, "free", 0)

	require.Equal(t, "disabled", extra["privacy_mode"])

	scope, ok := ExtractAccountModelScopeV2(extra)
	require.True(t, ok)
	require.NotNil(t, scope)
	require.Equal(t, AccountModelPolicyModeWhitelist, scope.PolicyMode)
	require.Len(t, scope.Entries, 4)
	require.Equal(t, "gpt-5.2", scope.Entries[0].DisplayModelID)
	require.Equal(t, "gpt-5.5", scope.Entries[3].TargetModelID)
}
