package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestEvaluatePublicModelCatalogDiscountOnceWindowBoundaries(t *testing.T) {
	policy := &PublicModelCatalogDiscountPolicy{
		Enabled:          true,
		ReductionPercent: 20,
		Timezone:         defaultPublicModelCatalogDiscountTimezone,
		Windows: []PublicModelCatalogDiscountWindow{{
			ID:      "launch",
			Type:    PublicModelCatalogDiscountWindowOnce,
			StartAt: "2026-06-01T00:00:00Z",
			EndAt:   "2026-06-01T01:00:00Z",
		}},
	}

	atStart := evaluatePublicModelCatalogDiscount(policy, mustParseTime(t, "2026-06-01T00:00:00Z"))
	require.True(t, atStart.Status.Active)
	require.Equal(t, "launch", atStart.Status.WindowID)

	beforeEndWithNanos := evaluatePublicModelCatalogDiscount(policy, time.Date(2026, 6, 1, 0, 59, 59, 1, time.UTC))
	require.False(t, beforeEndWithNanos.Status.Active)
	require.Equal(t, "2026-06-01T01:00:00Z", beforeEndWithNanos.Status.CompletedAt)

	atEnd := evaluatePublicModelCatalogDiscount(policy, mustParseTime(t, "2026-06-01T01:00:00Z"))
	require.False(t, atEnd.Status.Active)
}

func TestEvaluatePublicModelCatalogDiscountDailyAndCrossMidnight(t *testing.T) {
	policy := &PublicModelCatalogDiscountPolicy{
		Enabled:          true,
		ReductionPercent: 15,
		Timezone:         "Asia/Singapore",
		Windows: []PublicModelCatalogDiscountWindow{
			{
				ID:        "weekday",
				Type:      PublicModelCatalogDiscountWindowDaily,
				StartTime: "09:00:00",
				EndTime:   "10:00:00",
				Days:      []int{1},
			},
			{
				ID:        "late-night",
				Type:      PublicModelCatalogDiscountWindowDaily,
				StartTime: "23:00:00",
				EndTime:   "02:00:00",
				Days:      []int{6},
			},
		},
	}

	mondayMorning := evaluatePublicModelCatalogDiscount(policy, mustParseTime(t, "2026-06-01T01:30:00Z"))
	require.True(t, mondayMorning.Status.Active)
	require.Equal(t, "weekday", mondayMorning.Status.WindowID)

	sundayAfterMidnight := evaluatePublicModelCatalogDiscount(policy, mustParseTime(t, "2026-06-06T17:30:00Z"))
	require.True(t, sundayAfterMidnight.Status.Active)
	require.Equal(t, "late-night", sundayAfterMidnight.Status.WindowID)

	sundayAtEnd := evaluatePublicModelCatalogDiscount(policy, mustParseTime(t, "2026-06-06T18:00:00Z"))
	require.False(t, sundayAtEnd.Status.Active)
}

func TestNormalizePublicModelCatalogDiscountPolicyValidation(t *testing.T) {
	_, err := normalizePublicModelCatalogDiscountPolicy(&PublicModelCatalogDiscountPolicy{
		Enabled:          true,
		ReductionPercent: 0,
		Timezone:         "Asia/Singapore",
	})
	require.Error(t, err)

	normalized, err := normalizePublicModelCatalogDiscountPolicy(&PublicModelCatalogDiscountPolicy{
		Enabled:          false,
		ReductionPercent: 50,
		Timezone:         "Asia/Singapore",
		Windows: []PublicModelCatalogDiscountWindow{{
			Type:    PublicModelCatalogDiscountWindowOnce,
			StartAt: "2026-06-01T00:00:00Z",
			EndAt:   "2026-06-01T01:00:00Z",
		}},
	})
	require.NoError(t, err)
	require.NotNil(t, normalized)
	require.False(t, normalized.Enabled)
	require.Zero(t, normalized.ReductionPercent)
	require.Empty(t, normalized.Windows)
}

func TestApplyPublicModelCatalogDiscountRoundsUpToBillingPrecision(t *testing.T) {
	status := PublicModelCatalogDiscountStatus{Active: true, ReductionPercent: 20}
	display := applyPublicModelCatalogDiscountToPriceDisplay(PublicModelCatalogPriceDisplay{
		Primary: []PublicModelCatalogPriceEntry{{
			ID:         billingDiscountFieldInputPrice,
			Value:      0.000000011,
			Configured: true,
		}},
	}, status)

	require.Len(t, display.Primary, 1)
	require.Equal(t, 0.00000001, display.Primary[0].Value)
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return parsed
}
