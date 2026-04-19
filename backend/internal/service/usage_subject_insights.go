package service

import (
	"context"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
)

type UsageSubjectType string

const (
	UsageSubjectTypeAccount UsageSubjectType = "account"
	UsageSubjectTypeGroup   UsageSubjectType = "group"
	UsageSubjectTypeAPIKey  UsageSubjectType = "api_key"
)

type UsageSubjectInsightsQuery struct {
	SubjectType UsageSubjectType `json:"subject_type"`
	SubjectID   int64            `json:"subject_id"`
	StartTime   time.Time        `json:"start_time"`
	EndTime     time.Time        `json:"end_time"`
}

type UsageSubjectReference struct {
	Type      UsageSubjectType `json:"type"`
	ID        int64            `json:"id"`
	Name      string           `json:"name"`
	UserID    *int64           `json:"user_id,omitempty"`
	UserEmail string           `json:"user_email,omitempty"`
	GroupID   *int64           `json:"group_id,omitempty"`
	GroupName string           `json:"group_name,omitempty"`
}

type UsageSubjectSummary struct {
	TotalAccountCost   float64 `json:"total_account_cost"`
	TotalUserCost      float64 `json:"total_user_cost"`
	TotalStandardCost  float64 `json:"total_standard_cost"`
	TotalRequests      int64   `json:"total_requests"`
	TotalTokens        int64   `json:"total_tokens"`
	AvgDailyAccountCost float64 `json:"avg_daily_account_cost"`
	AvgDailyUserCost   float64 `json:"avg_daily_user_cost"`
	AvgDailyStandardCost float64 `json:"avg_daily_standard_cost"`
	AvgDailyRequests   float64 `json:"avg_daily_requests"`
	AvgDailyTokens     float64 `json:"avg_daily_tokens"`
	AvgDurationMs      float64 `json:"avg_duration_ms"`
	ActiveDays         int     `json:"active_days"`
	WindowDays         int     `json:"window_days"`
	Today              *UsageSubjectSummaryDay        `json:"today,omitempty"`
	HighestCostDay     *UsageSubjectSummaryCostDay    `json:"highest_cost_day,omitempty"`
	HighestRequestDay  *UsageSubjectSummaryRequestDay `json:"highest_request_day,omitempty"`
}

type UsageSubjectSummaryDay struct {
	Date             string  `json:"date"`
	AccountCost      float64 `json:"account_cost"`
	UserCost         float64 `json:"user_cost"`
	StandardCost     float64 `json:"standard_cost"`
	Requests         int64   `json:"requests"`
	Tokens           int64   `json:"tokens"`
}

type UsageSubjectSummaryCostDay struct {
	Date             string  `json:"date"`
	Label            string  `json:"label"`
	AccountCost      float64 `json:"account_cost"`
	UserCost         float64 `json:"user_cost"`
	StandardCost     float64 `json:"standard_cost"`
	Requests         int64   `json:"requests"`
}

type UsageSubjectSummaryRequestDay struct {
	Date             string  `json:"date"`
	Label            string  `json:"label"`
	Requests         int64   `json:"requests"`
	AccountCost      float64 `json:"account_cost"`
	UserCost         float64 `json:"user_cost"`
	StandardCost     float64 `json:"standard_cost"`
}

type UsageRequestPreviewCoverage struct {
	TotalRequests         int64   `json:"total_requests"`
	PreviewAvailableCount int64   `json:"preview_available_count"`
	PreviewAvailableRate  float64 `json:"preview_available_rate"`
	NormalizedCount       int64   `json:"normalized_count"`
	UpstreamRequestCount  int64   `json:"upstream_request_count"`
	UpstreamResponseCount int64   `json:"upstream_response_count"`
	GatewayResponseCount  int64   `json:"gateway_response_count"`
	ToolTraceCount        int64   `json:"tool_trace_count"`
}

type UsageSubjectInsights struct {
	Subject                UsageSubjectReference          `json:"subject"`
	Summary                UsageSubjectSummary            `json:"summary"`
	History                []usagestats.AccountUsageHistory `json:"history"`
	Models                 []usagestats.ModelStat        `json:"models"`
	Endpoints              []usagestats.EndpointStat     `json:"endpoints"`
	UpstreamEndpoints      []usagestats.EndpointStat     `json:"upstream_endpoints"`
	RequestPreviewCoverage UsageRequestPreviewCoverage   `json:"request_preview_coverage"`
}

type usageSubjectInsightsReader interface {
	GetSubjectUsageInsights(ctx context.Context, query UsageSubjectInsightsQuery) (*UsageSubjectInsights, error)
}

func (s *UsageService) GetSubjectUsageInsights(ctx context.Context, query UsageSubjectInsightsQuery) (*UsageSubjectInsights, error) {
	if query.SubjectID <= 0 {
		return nil, infraerrors.BadRequest("USAGE_SUBJECT_INVALID_ID", "subject_id must be greater than 0")
	}
	switch query.SubjectType {
	case UsageSubjectTypeAccount, UsageSubjectTypeGroup, UsageSubjectTypeAPIKey:
	default:
		return nil, infraerrors.BadRequest("USAGE_SUBJECT_INVALID_TYPE", "subject_type must be account, group, or api_key")
	}
	reader, ok := s.usageRepo.(usageSubjectInsightsReader)
	if !ok {
		return nil, infraerrors.ServiceUnavailable("USAGE_SUBJECT_INSIGHTS_UNAVAILABLE", "subject insights are not available")
	}
	if query.StartTime.IsZero() {
		query.StartTime = time.Now().UTC().AddDate(0, 0, -30)
	}
	if query.EndTime.IsZero() {
		query.EndTime = time.Now().UTC()
	}
	if query.EndTime.Before(query.StartTime) {
		query.StartTime, query.EndTime = query.EndTime, query.StartTime
	}
	return reader.GetSubjectUsageInsights(ctx, query)
}
