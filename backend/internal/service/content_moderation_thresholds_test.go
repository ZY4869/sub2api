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
