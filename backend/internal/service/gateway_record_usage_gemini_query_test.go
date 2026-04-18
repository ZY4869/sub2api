//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"github.com/stretchr/testify/require"
)

type gatewayRecordUsageQueryableRepoStub struct {
	UsageLogRepository

	logs            []UsageLog
	bestEffortCalls int
	lastLog         *UsageLog
}

func (s *gatewayRecordUsageQueryableRepoStub) CreateBestEffort(_ context.Context, log *UsageLog) error {
	copied := *log
	s.logs = append(s.logs, copied)
	s.bestEffortCalls++
	s.lastLog = &s.logs[len(s.logs)-1]
	return nil
}

func (s *gatewayRecordUsageQueryableRepoStub) Create(_ context.Context, log *UsageLog) (bool, error) {
	copied := *log
	s.logs = append(s.logs, copied)
	s.lastLog = &s.logs[len(s.logs)-1]
	return true, nil
}

func (s *gatewayRecordUsageQueryableRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, filters usagestats.UsageLogFilters) ([]UsageLog, *pagination.PaginationResult, error) {
	items := make([]UsageLog, 0, len(s.logs))
	for _, log := range s.logs {
		if filters.UserID > 0 && log.UserID != filters.UserID {
			continue
		}
		if filters.APIKeyID > 0 && log.APIKeyID != filters.APIKeyID {
			continue
		}
		if filters.StartTime != nil && log.CreatedAt.Before(*filters.StartTime) {
			continue
		}
		if filters.EndTime != nil && !log.CreatedAt.Before(*filters.EndTime) {
			continue
		}
		items = append(items, log)
	}

	page := params.Page
	if page <= 0 {
		page = 1
	}
	pageSize := params.PageSize
	if pageSize <= 0 {
		pageSize = len(items)
		if pageSize == 0 {
			pageSize = 1
		}
	}

	return items, &pagination.PaginationResult{
		Total:    int64(len(items)),
		Page:     page,
		PageSize: pageSize,
		Pages:    1,
	}, nil
}

func TestGatewayServiceRecordUsage_PersistsGeminiNativeUsageQueryableByAPIKeyAndDateRange(t *testing.T) {
	usageRepo := &gatewayRecordUsageQueryableRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})
	usageSvc := NewUsageService(usageRepo, nil, nil, nil)

	start := time.Now().Add(-time.Minute)
	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gemini-native-success",
			Usage: ClaudeUsage{
				InputTokens:  120,
				OutputTokens: 80,
			},
			Model:    "gemini-2.5-pro",
			Duration: 1500 * time.Millisecond,
		},
		APIKey: &APIKey{
			ID:      701,
			GroupID: i64p(81),
			Group:   &Group{ID: 81, RateMultiplier: 1},
		},
		User:             &User{ID: 601},
		Account:          &Account{ID: 501, Platform: PlatformGemini},
		InboundEndpoint:  "/v1beta/models/gemini-2.5-pro:generateContent",
		UpstreamEndpoint: "/v1beta/models/gemini-2.5-pro:generateContent",
		RequestBody:      []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	end := time.Now().Add(time.Minute)

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, int64(601), usageRepo.lastLog.UserID)
	require.Equal(t, int64(701), usageRepo.lastLog.APIKeyID)
	require.Equal(t, "gemini-native-success", usageRepo.lastLog.RequestID)
	require.NotNil(t, usageRepo.lastLog.InboundEndpoint)
	require.Equal(t, "/v1beta/models/gemini-2.5-pro:generateContent", *usageRepo.lastLog.InboundEndpoint)

	logs, result, err := usageSvc.ListWithFilters(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{
		UserID:    601,
		APIKeyID:  701,
		StartTime: &start,
		EndTime:   &end,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, logs, 1)
	require.Equal(t, "gemini-native-success", logs[0].RequestID)
	require.NotNil(t, logs[0].InboundEndpoint)
	require.Equal(t, "/v1beta/models/gemini-2.5-pro:generateContent", *logs[0].InboundEndpoint)
}

func TestGatewayServiceRecordUsage_PersistsGeminiPassthroughUsageQueryableByAPIKeyAndDateRange(t *testing.T) {
	usageRepo := &gatewayRecordUsageQueryableRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})
	usageSvc := NewUsageService(usageRepo, nil, nil, nil)

	inboundEndpoint := "/v1/projects/demo/locations/us-central1/publishers/google/models/gemini-2.5-pro:generateContent"
	upstreamEndpoint := EndpointVertexSyncModels
	start := time.Now().Add(-time.Minute)
	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "gemini-passthrough-success",
			Usage: ClaudeUsage{
				InputTokens:  64,
				OutputTokens: 32,
			},
			Model:    "gemini-2.5-pro",
			Duration: time.Second,
		},
		APIKey: &APIKey{
			ID:      702,
			GroupID: i64p(82),
			Group:   &Group{ID: 82, RateMultiplier: 1},
		},
		User:             &User{ID: 602},
		Account:          &Account{ID: 502, Platform: PlatformGemini},
		InboundEndpoint:  inboundEndpoint,
		UpstreamEndpoint: upstreamEndpoint,
		RequestBody:      []byte(`{"contents":[{"parts":[{"text":"hi"}]}]}`),
	})
	end := time.Now().Add(time.Minute)

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, int64(602), usageRepo.lastLog.UserID)
	require.Equal(t, int64(702), usageRepo.lastLog.APIKeyID)
	require.Equal(t, "gemini-passthrough-success", usageRepo.lastLog.RequestID)
	require.NotNil(t, usageRepo.lastLog.InboundEndpoint)
	require.Equal(t, inboundEndpoint, *usageRepo.lastLog.InboundEndpoint)
	require.NotNil(t, usageRepo.lastLog.UpstreamEndpoint)
	require.Equal(t, upstreamEndpoint, *usageRepo.lastLog.UpstreamEndpoint)

	logs, result, err := usageSvc.ListWithFilters(context.Background(), pagination.PaginationParams{Page: 1, PageSize: 20}, usagestats.UsageLogFilters{
		UserID:    602,
		APIKeyID:  702,
		StartTime: &start,
		EndTime:   &end,
	})
	require.NoError(t, err)
	require.NotNil(t, result)
	require.Equal(t, int64(1), result.Total)
	require.Len(t, logs, 1)
	require.Equal(t, "gemini-passthrough-success", logs[0].RequestID)
	require.NotNil(t, logs[0].InboundEndpoint)
	require.Equal(t, inboundEndpoint, *logs[0].InboundEndpoint)
	require.NotNil(t, logs[0].UpstreamEndpoint)
	require.Equal(t, upstreamEndpoint, *logs[0].UpstreamEndpoint)
}
