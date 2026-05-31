package service

import (
	"testing"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestEvaluateTimeAccessPolicy_OvernightWindow(t *testing.T) {
	policy := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1},
			Start: "22:00",
			End:   "02:00",
		}},
	}

	mondayLate := time.Date(2026, 6, 1, 23, 0, 0, 0, time.FixedZone("SGT", 8*3600))
	tuesdayEarly := time.Date(2026, 6, 2, 1, 30, 0, 0, time.FixedZone("SGT", 8*3600))
	tuesdayNoon := time.Date(2026, 6, 2, 12, 0, 0, 0, time.FixedZone("SGT", 8*3600))

	require.True(t, EvaluateTimeAccessPolicy(policy, mondayLate).Allowed)
	require.True(t, EvaluateTimeAccessPolicy(policy, tuesdayEarly).Allowed)
	require.False(t, EvaluateTimeAccessPolicy(policy, tuesdayNoon).Allowed)
}

func TestEvaluateTimeAccessPolicy_InvalidTimezoneDenied(t *testing.T) {
	policy := &TimeAccessPolicy{
		Enabled:       true,
		Timezone:      "Mars/Base",
		WeeklyWindows: []TimeAccessWindow{{Days: []int{1}, Start: "08:00", End: "20:00"}},
	}

	result := EvaluateTimeAccessPolicy(policy, time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC))

	require.False(t, result.Allowed)
	require.Equal(t, TimeAccessDecisionInvalidPolicy, result.Reason)
}

func TestNormalizeTimeAccessPolicy_DailyAllowedMinutesCapsConfiguredWindows(t *testing.T) {
	limit := 480
	policy := &TimeAccessPolicy{
		Enabled:             true,
		Timezone:            "Asia/Singapore",
		DailyAllowedMinutes: &limit,
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1, 2, 3, 4, 5},
			Start: "08:00",
			End:   "20:00",
		}},
	}

	_, err := NormalizeTimeAccessPolicy(policy)

	require.Error(t, err)
	require.Contains(t, err.Error(), "weekly_windows exceed daily_allowed_minutes")
}

func TestNormalizeTimeAccessPolicy_DailyAllowedMinutesUsesMinutePrecision(t *testing.T) {
	limit := 480
	policy := &TimeAccessPolicy{
		Enabled:             true,
		Timezone:            "Asia/Singapore",
		DailyAllowedMinutes: &limit,
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1},
			Start: "09:00",
			End:   "17:01",
		}},
	}

	_, err := NormalizeTimeAccessPolicy(policy)

	require.Error(t, err)
	require.Contains(t, err.Error(), "weekly_windows exceed daily_allowed_minutes")
}

func TestEvaluateTimeAccessPolicy_WindowBoundaryEndExclusive(t *testing.T) {
	policy := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1},
			Start: "08:00",
			End:   "20:00",
		}},
	}

	atStart := time.Date(2026, 6, 1, 8, 0, 0, 0, time.FixedZone("SGT", 8*3600))
	atEnd := time.Date(2026, 6, 1, 20, 0, 0, 0, time.FixedZone("SGT", 8*3600))

	require.True(t, EvaluateTimeAccessPolicy(policy, atStart).Allowed)
	require.False(t, EvaluateTimeAccessPolicy(policy, atEnd).Allowed)
}

func TestValidateTimeAccessSubset_UserHardLimit(t *testing.T) {
	parent := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1, 2, 3, 4, 5},
			Start: "08:00",
			End:   "20:00",
		}},
	}
	childOK := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1, 2, 3},
			Start: "09:00",
			End:   "17:00",
		}},
	}
	childTooWide := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1},
			Start: "07:00",
			End:   "17:00",
		}},
	}

	require.NoError(t, ValidateTimeAccessSubset(childOK, parent))
	require.Error(t, ValidateTimeAccessSubset(childTooWide, parent))
}

func TestValidateTimeAccessSubset_UsesMinutePrecision(t *testing.T) {
	parent := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1},
			Start: "08:00",
			End:   "20:00",
		}},
	}
	child := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{1},
			Start: "07:59",
			End:   "08:01",
		}},
	}

	err := ValidateTimeAccessSubset(child, parent)

	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeds user hard limit")
}

func TestValidateTimeAccessSubset_OvernightAcrossWeekBoundary(t *testing.T) {
	parent := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{0},
			Start: "22:00",
			End:   "02:00",
		}},
	}
	childOK := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{0},
			Start: "23:00",
			End:   "01:00",
		}},
	}
	childTooWide := &TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Asia/Singapore",
		WeeklyWindows: []TimeAccessWindow{{
			Days:  []int{0},
			Start: "21:59",
			End:   "01:00",
		}},
	}

	require.NoError(t, ValidateTimeAccessSubset(childOK, parent))
	require.Error(t, ValidateTimeAccessSubset(childTooWide, parent))
}

func TestEvaluateEffectiveTimeAccess_StartsAtAndPolicies(t *testing.T) {
	now := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	startsAt := now.Add(time.Hour)
	key := &APIKey{
		StartsAt: &startsAt,
		User:     &User{},
	}

	result := key.EvaluateTimeAccess(now)
	require.False(t, result.Allowed)
	require.Equal(t, TimeAccessDecisionNotBefore, result.Reason)
}

func TestTimeAccessPolicyInputErrorIsFriendlyBadRequest(t *testing.T) {
	_, rawErr := NormalizeTimeAccessPolicy(&TimeAccessPolicy{
		Enabled:  true,
		Timezone: "Mars/Base",
	})
	require.Error(t, rawErr)

	err := timeAccessPolicyInputError(rawErr)
	appErr := infraerrors.FromError(err)

	require.Equal(t, int32(400), appErr.Code)
	require.Equal(t, "TIME_ACCESS_POLICY_INVALID", appErr.Reason)
	require.Equal(t, "时间访问策略无效，请检查时区、时间窗口和每日上限", appErr.Message)
	require.NotContains(t, appErr.Message, "Mars/Base")
}
