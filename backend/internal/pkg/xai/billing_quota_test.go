package xai

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBuildBillingSummaryParsesWeeklyAndMonthlyPayloads(t *testing.T) {
	weeklyPayload := []byte(`{
		"config": {
			"currentPeriod": {"type": "weekly", "start": "2026-07-13T00:00:00Z", "end": "2026-07-20T00:00:00Z"},
			"creditUsagePercent": 12.5,
			"productUsage": [{"product": "grok-4.5", "usagePercent": 10}]
		}
	}`)
	monthlyPayload := []byte(`{
		"config": {
			"monthlyLimit": {"val": "15000"},
			"used": {"val": 3000},
			"billingPeriodStart": "2026-07-01T00:00:00Z",
			"billingPeriodEnd": "2026-08-01T00:00:00Z"
		}
	}`)

	weekly, err := ParseBillingPayload(weeklyPayload)
	require.NoError(t, err)
	monthly, err := ParseBillingPayload(monthlyPayload)
	require.NoError(t, err)

	merged := MergeBillingProbeResult(nil, BuildBillingSummary(weekly.Config), BuildBillingSummary(monthly.Config), true, true)
	require.NotNil(t, merged)
	require.Equal(t, "weekly", merged.PeriodType)
	require.NotNil(t, merged.UsagePercent)
	require.Equal(t, 12.5, *merged.UsagePercent)
	require.Len(t, merged.ProductUsage, 1)
	require.Equal(t, "grok-4.5", merged.ProductUsage[0].Product)
	require.NotNil(t, merged.MonthlyLimitCents)
	require.Equal(t, float64(SuperGrokLimitCents), *merged.MonthlyLimitCents)
	require.NotNil(t, merged.UsedPercent)
	require.Equal(t, float64(20), *merged.UsedPercent)
	require.Equal(t, "SuperGrok", merged.Plan)
	require.False(t, merged.Partial)
}

func TestObserveQuotaHeadersParsesAllowlistedHeaders(t *testing.T) {
	headers := http.Header{}
	headers.Set("x-ratelimit-limit-requests", "100")
	headers.Set("x-ratelimit-remaining-requests", "40")
	headers.Set("x-ratelimit-reset-requests", "1784073600000")
	headers.Set("x-ratelimit-limit-tokens", "1000")
	headers.Set("x-ratelimit-remaining-tokens", "700")
	headers.Set("retry-after", "15")
	headers.Set("xai-subscription-tier", "SuperGrok")
	headers.Set("x-entitlement-status", "active")
	headers.Set("authorization", "Bearer should-not-be-copied")

	snapshot := ObserveQuotaHeaders(headers, http.StatusTooManyRequests, "active_probe")
	require.NotNil(t, snapshot)
	require.True(t, snapshot.HeadersObserved)
	require.Equal(t, http.StatusTooManyRequests, snapshot.StatusCode)
	require.NotNil(t, snapshot.Requests)
	require.EqualValues(t, 100, *snapshot.Requests.Limit)
	require.EqualValues(t, 40, *snapshot.Requests.Remaining)
	require.NotEmpty(t, snapshot.Requests.ResetAt)
	require.NotNil(t, snapshot.Tokens)
	require.EqualValues(t, 700, *snapshot.Tokens.Remaining)
	require.NotNil(t, snapshot.RetryAfterSeconds)
	require.Equal(t, 15, *snapshot.RetryAfterSeconds)
	require.Equal(t, "SuperGrok", snapshot.SubscriptionTier)
	require.Equal(t, "active", snapshot.EntitlementStatus)
	require.NotContains(t, snapshot.Headers, "authorization")

	encoded, err := json.Marshal(snapshot)
	require.NoError(t, err)
	require.NotContains(t, string(encoded), "should-not-be-copied")
}
