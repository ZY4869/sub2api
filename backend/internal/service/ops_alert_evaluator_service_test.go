//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var _ OpsRepository = (*stubOpsRepo)(nil)

type stubOpsRepo struct {
	OpsRepository
	overview      *OpsDashboardOverview
	err           error
	overviewCalls int
	overviewModes []OpsQueryMode
}

func (s *stubOpsRepo) GetDashboardOverview(ctx context.Context, filter *OpsDashboardFilter) (*OpsDashboardOverview, error) {
	s.overviewCalls++
	if filter != nil {
		s.overviewModes = append(s.overviewModes, filter.QueryMode)
	}
	if s.err != nil {
		return nil, s.err
	}
	if s.overview != nil {
		return s.overview, nil
	}
	return &OpsDashboardOverview{}, nil
}

func TestComputeGroupAvailableRatio(t *testing.T) {
	t.Parallel()

	t.Run("正常情况: 10个账号, 8个可用 = 80%", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  10,
			AvailableCount: 8,
		})
		require.InDelta(t, 80.0, got, 0.0001)
	})

	t.Run("边界情况: TotalAccounts = 0 应返回 0", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  0,
			AvailableCount: 8,
		})
		require.Equal(t, 0.0, got)
	})

	t.Run("边界情况: AvailableCount = 0 应返回 0%", func(t *testing.T) {
		t.Parallel()

		got := computeGroupAvailableRatio(&GroupAvailability{
			TotalAccounts:  10,
			AvailableCount: 0,
		})
		require.Equal(t, 0.0, got)
	})
}

func TestCountAccountsByCondition(t *testing.T) {
	t.Parallel()

	t.Run("测试限流账号统计: acc.IsRateLimited", func(t *testing.T) {
		t.Parallel()

		accounts := map[int64]*AccountAvailability{
			1: {IsRateLimited: true},
			2: {IsRateLimited: false},
			3: {IsRateLimited: true},
		}

		got := countAccountsByCondition(accounts, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		})
		require.Equal(t, int64(2), got)
	})

	t.Run("测试错误账号统计（排除临时不可调度）: acc.HasError && acc.TempUnschedulableUntil == nil", func(t *testing.T) {
		t.Parallel()

		until := time.Now().UTC().Add(5 * time.Minute)
		accounts := map[int64]*AccountAvailability{
			1: {HasError: true},
			2: {HasError: true, TempUnschedulableUntil: &until},
			3: {HasError: false},
		}

		got := countAccountsByCondition(accounts, func(acc *AccountAvailability) bool {
			return acc.HasError && acc.TempUnschedulableUntil == nil
		})
		require.Equal(t, int64(1), got)
	})

	t.Run("边界情况: 空 map 应返回 0", func(t *testing.T) {
		t.Parallel()

		got := countAccountsByCondition(map[int64]*AccountAvailability{}, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		})
		require.Equal(t, int64(0), got)
	})
}

func TestComputeRuleMetricNewIndicators(t *testing.T) {
	t.Parallel()

	groupID := int64(101)
	platform := "openai"

	availability := &OpsAccountAvailability{
		Group: &GroupAvailability{
			GroupID:        groupID,
			TotalAccounts:  10,
			AvailableCount: 8,
		},
		Accounts: map[int64]*AccountAvailability{
			1: {IsRateLimited: true},
			2: {IsRateLimited: true},
			3: {HasError: true},
			4: {HasError: true, TempUnschedulableUntil: timePtr(time.Now().UTC().Add(2 * time.Minute))},
			5: {HasError: false, IsRateLimited: false},
		},
	}

	opsService := &OpsService{
		getAccountAvailability: func(_ context.Context, _ string, _ *int64) (*OpsAccountAvailability, error) {
			return availability, nil
		},
	}

	svc := &OpsAlertEvaluatorService{
		opsService: opsService,
		opsRepo:    &stubOpsRepo{overview: &OpsDashboardOverview{}},
	}

	start := time.Now().UTC().Add(-5 * time.Minute)
	end := time.Now().UTC()
	ctx := context.Background()

	tests := []struct {
		name       string
		metricType string
		groupID    *int64
		wantValue  float64
		wantOK     bool
	}{
		{
			name:       "group_available_accounts",
			metricType: "group_available_accounts",
			groupID:    &groupID,
			wantValue:  8,
			wantOK:     true,
		},
		{
			name:       "group_available_ratio",
			metricType: "group_available_ratio",
			groupID:    &groupID,
			wantValue:  80.0,
			wantOK:     true,
		},
		{
			name:       "account_rate_limited_count",
			metricType: "account_rate_limited_count",
			groupID:    nil,
			wantValue:  2,
			wantOK:     true,
		},
		{
			name:       "account_error_count",
			metricType: "account_error_count",
			groupID:    nil,
			wantValue:  1,
			wantOK:     true,
		},
		{
			name:       "group_available_accounts without group_id returns false",
			metricType: "group_available_accounts",
			groupID:    nil,
			wantValue:  0,
			wantOK:     false,
		},
		{
			name:       "group_available_ratio without group_id returns false",
			metricType: "group_available_ratio",
			groupID:    nil,
			wantValue:  0,
			wantOK:     false,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			rule := &OpsAlertRule{
				MetricType: tt.metricType,
			}
			gotValue, gotOK := svc.computeRuleMetric(ctx, rule, nil, start, end, platform, tt.groupID, newOpsAlertEvaluationCache())
			require.Equal(t, tt.wantOK, gotOK)
			if !tt.wantOK {
				return
			}
			require.InDelta(t, tt.wantValue, gotValue, 0.0001)
		})
	}
}

func TestComputeRuleMetricCachesOverviewPerEvaluationCycle(t *testing.T) {
	t.Parallel()

	groupID := int64(101)
	start := time.Now().UTC().Add(-5 * time.Minute)
	end := time.Now().UTC()
	repo := &stubOpsRepo{
		overview: &OpsDashboardOverview{
			RequestCountSLA:   10,
			SLA:               0.9,
			ErrorRate:         0.1,
			UpstreamErrorRate: 0.02,
		},
	}

	svc := &OpsAlertEvaluatorService{opsRepo: repo}
	cache := newOpsAlertEvaluationCache()

	tests := []struct {
		metricType string
		wantValue  float64
	}{
		{metricType: "success_rate", wantValue: 90},
		{metricType: "error_rate", wantValue: 10},
		{metricType: "upstream_error_rate", wantValue: 2},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.metricType, func(t *testing.T) {
			got, ok := svc.computeRuleMetric(
				context.Background(),
				&OpsAlertRule{MetricType: tt.metricType},
				nil,
				start,
				end,
				"openai",
				&groupID,
				cache,
			)
			require.True(t, ok)
			require.InDelta(t, tt.wantValue, got, 0.0001)
		})
	}

	require.Equal(t, 1, repo.overviewCalls)
	require.Equal(t, []OpsQueryMode{OpsQueryModeAuto}, repo.overviewModes)
	require.Equal(t, 1, cache.stats.OverviewMisses)
	require.Equal(t, 2, cache.stats.OverviewHits)
}

func TestComputeRuleMetricCachesAvailabilityPerEvaluationCycle(t *testing.T) {
	t.Parallel()

	groupID := int64(101)
	calls := 0
	availability := &OpsAccountAvailability{
		Group: &GroupAvailability{
			GroupID:        groupID,
			TotalAccounts:  10,
			AvailableCount: 8,
			RateLimitCount: 2,
		},
		Accounts: map[int64]*AccountAvailability{
			1: {IsRateLimited: true},
			2: {HasError: true},
			3: {IsOverloaded: true},
		},
	}

	svc := &OpsAlertEvaluatorService{
		opsService: &OpsService{
			getAccountAvailability: func(_ context.Context, _ string, gotGroupID *int64) (*OpsAccountAvailability, error) {
				calls++
				require.NotNil(t, gotGroupID)
				require.Equal(t, groupID, *gotGroupID)
				return availability, nil
			},
		},
	}

	cache := newOpsAlertEvaluationCache()
	tests := []struct {
		metricType string
		wantValue  float64
	}{
		{metricType: "group_available_ratio", wantValue: 80},
		{metricType: "account_rate_limited_count", wantValue: 1},
		{metricType: "overload_account_count", wantValue: 1},
	}

	for _, tt := range tests {
		got, ok := svc.computeRuleMetric(
			context.Background(),
			&OpsAlertRule{MetricType: tt.metricType},
			nil,
			time.Now().UTC().Add(-5*time.Minute),
			time.Now().UTC(),
			"openai",
			&groupID,
			cache,
		)
		require.True(t, ok)
		require.InDelta(t, tt.wantValue, got, 0.0001)
	}

	require.Equal(t, 1, calls)
	require.Equal(t, 1, cache.stats.AvailabilityMisses)
	require.Equal(t, 2, cache.stats.AvailabilityHits)
}
