package service

import (
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestCheckUserPlatformQuotaAllowed(t *testing.T) {
	now := time.Date(2026, 5, 26, 8, 0, 0, 0, time.UTC)
	start := now.Add(-time.Hour)
	limit := 10.0

	require.NoError(t, CheckUserPlatformQuotaAllowed(nil, "openai", now))
	require.NoError(t, CheckUserPlatformQuotaAllowed([]UserPlatformQuota{{
		Platform:      "openai",
		DailyUsageUSD: 99,
	}}, "openai", now))
	require.NoError(t, CheckUserPlatformQuotaAllowed([]UserPlatformQuota{{
		Platform:         "openai",
		DailyLimitUSD:    &limit,
		DailyUsageUSD:    limit,
		DailyWindowStart: ptrUserPlatformQuotaTime(now.Add(-25 * time.Hour)),
	}}, "openai", now))

	err := CheckUserPlatformQuotaAllowed([]UserPlatformQuota{{
		Platform:         "openai",
		DailyLimitUSD:    &limit,
		DailyUsageUSD:    limit,
		DailyWindowStart: &start,
	}}, "openai", now)

	require.ErrorIs(t, err, ErrUserPlatformQuotaExceeded)
	require.Equal(t, 429, infraerrors.Code(err))
	require.Equal(t, "USER_PLATFORM_QUOTA_EXCEEDED", infraerrors.Reason(err))
}

func TestNormalizeUserPlatformQuotaInputs(t *testing.T) {
	daily := 12.34567890123
	weekly := 0.0
	monthly := -1.0

	items, err := NormalizeUserPlatformQuotaInputs([]UserPlatformQuotaInput{
		{Platform: " OpenAI ", DailyLimitUSD: &daily, WeeklyLimitUSD: &weekly},
		{Platform: "openai", DailyLimitUSD: ptrFloat(2)},
	})
	require.NoError(t, err)
	require.Len(t, items, 1)
	require.Equal(t, "openai", items[0].Platform)
	require.InDelta(t, 2, *items[0].DailyLimitUSD, 1e-12)
	require.Nil(t, items[0].WeeklyLimitUSD)

	_, err = NormalizeUserPlatformQuotaInputs([]UserPlatformQuotaInput{
		{Platform: "openai", MonthlyLimitUSD: &monthly},
	})
	require.Error(t, err)
	require.Equal(t, "INVALID_QUOTA_LIMIT", infraerrors.Reason(err))
}

func TestBuildUserPlatformQuotaViewsResetsExpiredWindow(t *testing.T) {
	now := time.Date(2026, 5, 26, 8, 0, 0, 0, time.UTC)
	start := now.Add(-2 * time.Hour)
	expired := now.Add(-8 * 24 * time.Hour)
	daily := 10.0
	weekly := 20.0

	views := BuildUserPlatformQuotaViews([]UserPlatformQuota{{
		Platform:          "openai",
		DailyLimitUSD:     &daily,
		WeeklyLimitUSD:    &weekly,
		DailyUsageUSD:     3,
		WeeklyUsageUSD:    19,
		DailyWindowStart:  &start,
		WeeklyWindowStart: &expired,
	}}, now)

	require.Len(t, views, 1)
	require.Equal(t, "openai", views[0].Platform)
	require.InDelta(t, 3, views[0].Daily.UsageUSD, 1e-12)
	require.NotNil(t, views[0].Daily.ResetAt)
	require.Zero(t, views[0].Weekly.UsageUSD)
	require.Nil(t, views[0].Weekly.WindowStart)
	require.Nil(t, views[0].Weekly.ResetAt)
}

func ptrFloat(value float64) *float64 {
	return &value
}

func ptrUserPlatformQuotaTime(value time.Time) *time.Time {
	return &value
}
