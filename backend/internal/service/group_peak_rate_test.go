package service

import (
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/stretchr/testify/require"
)

func TestNormalizeGroupPeakRateConfig(t *testing.T) {
	t.Run("standard group clears peak config", func(t *testing.T) {
		enabled, start, end, multiplier, err := NormalizeGroupPeakRateConfig(SubscriptionTypeStandard, true, "09:00", "18:00", 2)
		require.NoError(t, err)
		require.False(t, enabled)
		require.Empty(t, start)
		require.Empty(t, end)
		require.Equal(t, 1.0, multiplier)
	})

	t.Run("subscription group accepts trimmed HHMM window", func(t *testing.T) {
		enabled, start, end, multiplier, err := NormalizeGroupPeakRateConfig(SubscriptionTypeSubscription, true, " 09:00 ", "18:00", 1.75)
		require.NoError(t, err)
		require.True(t, enabled)
		require.Equal(t, "09:00", start)
		require.Equal(t, "18:00", end)
		require.Equal(t, 1.75, multiplier)
	})

	t.Run("rejects invalid time", func(t *testing.T) {
		_, _, _, _, err := NormalizeGroupPeakRateConfig(SubscriptionTypeSubscription, true, "9:00", "18:00", 1.5)
		require.ErrorContains(t, err, "invalid peak_start")
	})

	t.Run("rejects non increasing window", func(t *testing.T) {
		_, _, _, _, err := NormalizeGroupPeakRateConfig(SubscriptionTypeSubscription, true, "18:00", "09:00", 1.5)
		require.ErrorContains(t, err, "peak_start must be earlier than peak_end")
	})
}

func TestGroupEffectiveTokenRateMultiplierAt(t *testing.T) {
	group := &Group{
		SubscriptionType:   SubscriptionTypeSubscription,
		PeakRateEnabled:    true,
		PeakStart:          "09:00",
		PeakEnd:            "18:00",
		PeakRateMultiplier: 2.5,
	}

	require.True(t, group.IsPeakRateActiveAt(time.Date(2026, 7, 3, 10, 0, 0, 0, timezone.Location())))
	require.Equal(t, 3.0, group.EffectiveTokenRateMultiplierAt(1.2, time.Date(2026, 7, 3, 10, 0, 0, 0, timezone.Location())))
	require.False(t, group.IsPeakRateActiveAt(time.Date(2026, 7, 3, 18, 0, 0, 0, timezone.Location())))
	require.Equal(t, 1.2, group.EffectiveTokenRateMultiplierAt(1.2, time.Date(2026, 7, 3, 18, 0, 0, 0, timezone.Location())))

	group.SubscriptionType = SubscriptionTypeStandard
	require.False(t, group.IsPeakRateActiveAt(time.Date(2026, 7, 3, 10, 0, 0, 0, timezone.Location())))
	require.Equal(t, 1.2, group.EffectiveTokenRateMultiplierAt(1.2, time.Date(2026, 7, 3, 10, 0, 0, 0, timezone.Location())))
}
