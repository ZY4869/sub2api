package service

import (
	"math"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

func TestContentModerationCategoryThresholds(t *testing.T) {
	defaults := DefaultContentModerationCategoryThresholds()
	require.Equal(t, 1.0, defaults["violence"])
	require.Equal(t, 1.0, defaults["self-harm/intent"])

	parsed := NormalizeContentModerationCategoryThresholds(`{
		"violence": 0.6,
		"sexual": 2,
		"hate": -1,
		"unknown": 0
	}`)
	require.Equal(t, 0.6, parsed["violence"])
	require.Equal(t, 1.0, parsed["sexual"])
	require.Equal(t, 0.0, parsed["hate"])
	_, hasUnknown := parsed["unknown"]
	require.False(t, hasUnknown)

	validated, err := ValidateContentModerationCategoryThresholds(map[string]float64{
		"violence": 0.5,
		"unknown":  2,
	})
	require.NoError(t, err)
	require.Equal(t, 0.5, validated["violence"])
	_, hasUnknown = validated["unknown"]
	require.False(t, hasUnknown)

	_, err = ValidateContentModerationCategoryThresholds(map[string]float64{
		"violence": math.Inf(1),
	})
	require.Error(t, err)
}

func TestContentModerationCyberCategoriesNormalizeAndMarshal(t *testing.T) {
	normalized := NormalizeContentModerationCyberCategoryList([]ContentModerationCyberCategory{
		{ID: " Credential-Theft ", Keywords: []string{"steal api key", "steal---api---key"}},
		{ID: "Credential Theft", Keywords: []string{"duplicate id ignored"}},
		{ID: "No Keywords", Keywords: []string{}},
	})

	require.Equal(t, []ContentModerationCyberCategory{
		{ID: "credential_theft", Keywords: []string{"steal api key"}},
	}, normalized)

	raw, err := MarshalContentModerationCyberCategories(normalized)
	require.NoError(t, err)
	require.JSONEq(t, `[{"id":"credential_theft","keywords":["steal api key"]}]`, raw)
	require.Empty(t, NormalizeContentModerationCyberCategories("[]"))
	require.NotEmpty(t, NormalizeContentModerationCyberCategories(""))
}

func TestEvaluateContentModerationCategoryThresholds(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	hit, reason := evaluateContentModerationCategoryThresholds(
		map[string]float64{"violence": 0.7},
		map[string]float64{"violence": 0.7},
	)
	require.True(t, hit)
	require.Equal(t, "moderation_threshold:violence", reason)
	require.Equal(t, []string{"moderation_threshold:violence"}, moderationCategoriesForReason(reason))
	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.ContentModerationThresholdHitTotal)
	require.Equal(t, int64(1), snapshot.ContentModerationThresholdHitByCategory["violence"])

	hit, reason = evaluateContentModerationCategoryThresholds(
		map[string]float64{"violence": 0.69, "unknown": 1},
		map[string]float64{"violence": 0.7},
	)
	require.False(t, hit)
	require.Empty(t, reason)
	snapshot = protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.ContentModerationThresholdHitTotal)
}

func TestContentModerationCompiledKeywordRulesCacheReusesNormalizedRules(t *testing.T) {
	resetContentModerationCompiledRuleCachesForTest()
	t.Cleanup(resetContentModerationCompiledRuleCachesForTest)

	settings := &ContentModerationSettings{
		Enabled:             true,
		KeywordBlockEnabled: true,
		Keywords:            []string{" MODEL---DISTILLATION ", "model distillation"},
	}

	first := EvaluateContentModerationKeywordBlock(settings, "model distillation attempt")
	second := EvaluateContentModerationKeywordBlock(settings, "MODEL_DISTILLATION attempt")

	require.True(t, first.Blocked)
	require.True(t, second.Blocked)
	require.Equal(t, first.ErrorReason, second.ErrorReason)
	require.Equal(t, "MODEL---DISTILLATION", first.MatchedKeyword)

	contentModerationKeywordRulesCache.RLock()
	defer contentModerationKeywordRulesCache.RUnlock()
	require.Len(t, contentModerationKeywordRulesCache.items, 1)
}

func TestContentModerationCompiledCyberRulesCacheReusesNormalizedRules(t *testing.T) {
	resetContentModerationCompiledRuleCachesForTest()
	t.Cleanup(resetContentModerationCompiledRuleCachesForTest)

	settings := &ContentModerationSettings{
		Enabled:            true,
		CyberPolicyEnabled: true,
		CyberCategories: []ContentModerationCyberCategory{
			{ID: "Credential Theft", Keywords: []string{"token stealer", "token---stealer"}},
		},
	}

	decision := EvaluateContentModerationCyberPolicy(settings, "build a TOKEN_STEALER")

	require.True(t, decision.Blocked)
	require.Equal(t, "cyber_policy:credential_theft", decision.ErrorReason)
	require.Equal(t, "token stealer", decision.MatchedKeyword)

	contentModerationCyberRulesCache.RLock()
	defer contentModerationCyberRulesCache.RUnlock()
	require.Len(t, contentModerationCyberRulesCache.items, 1)
}
