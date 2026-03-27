package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

type usageLogFailureRepoStub struct {
	UsageLogRepository

	bestEffortCalls int
	lastLog         *UsageLog
}

func (s *usageLogFailureRepoStub) CreateBestEffort(ctx context.Context, log *UsageLog) error {
	s.bestEffortCalls++
	s.lastLog = log
	return nil
}

func TestBuildFailedUsageLogBase_SanitizesAndMarksFailed(t *testing.T) {
	groupID := int64(10)
	subscriptionID := int64(20)
	ctx := context.WithValue(context.Background(), ctxkey.RequestID, "failed-usage-local")

	log := buildFailedUsageLogBase(
		ctx,
		&APIKey{ID: 1, GroupID: &groupID, Group: &Group{ID: groupID}},
		&User{ID: 2},
		&Account{ID: 3},
		&UserSubscription{ID: subscriptionID},
		1.25,
		&RecordFailedUsageInput{
			Model:            " gemini-2.5-pro ",
			UpstreamModel:    " gemini-2.5-pro-preview ",
			InboundEndpoint:  " /v1/messages ",
			UpstreamEndpoint: " /v1beta/models/gemini-2.5-pro:generateContent ",
			HTTPStatus:       429,
			ErrorCode:        strings.Repeat("x", failedUsageErrorCodeMaxLen+8),
			ErrorMessage:     `authorization=Bearer-secret-token x-api-key=super-secret`,
			SimulatedClient:  GatewayClientProfileGeminiCLI,
			Stream:           true,
			Duration:         1500 * time.Millisecond,
		},
	)

	require.NotNil(t, log)
	require.Equal(t, UsageLogStatusFailed, log.Status)
	require.Equal(t, "local:failed-usage-local", log.RequestID)
	require.Equal(t, "gemini-2.5-pro", log.Model)
	require.NotNil(t, log.UpstreamModel)
	require.Equal(t, " gemini-2.5-pro-preview ", *log.UpstreamModel)
	require.NotNil(t, log.InboundEndpoint)
	require.Equal(t, "/v1/messages", *log.InboundEndpoint)
	require.NotNil(t, log.UpstreamEndpoint)
	require.Equal(t, "/v1beta/models/gemini-2.5-pro:generateContent", *log.UpstreamEndpoint)
	require.NotNil(t, log.HTTPStatus)
	require.Equal(t, 429, *log.HTTPStatus)
	require.NotNil(t, log.ErrorCode)
	require.Len(t, *log.ErrorCode, failedUsageErrorCodeMaxLen)
	require.NotNil(t, log.ErrorMessage)
	require.NotContains(t, *log.ErrorMessage, "Bearer-secret-token")
	require.NotContains(t, *log.ErrorMessage, "super-secret")
	require.NotNil(t, log.SimulatedClient)
	require.Equal(t, GatewayClientProfileGeminiCLI, *log.SimulatedClient)
	require.Equal(t, RequestTypeStream, log.RequestType)
	require.Equal(t, groupID, *log.GroupID)
	require.Equal(t, subscriptionID, *log.SubscriptionID)
	require.Equal(t, 0.0, log.ActualCost)
}

func TestGatewayServiceRecordFailedUsage_WritesBestEffortUsageLog(t *testing.T) {
	repo := &usageLogFailureRepoStub{}
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1.1
	svc := &GatewayService{
		cfg:             cfg,
		usageLogRepo:    repo,
		deferredService: &DeferredService{},
	}

	ctx := WithThinkingEnabled(context.Background(), true, false)
	err := svc.RecordFailedUsage(ctx, &RecordFailedUsageInput{
		APIKey: &APIKey{
			ID: 11,
		},
		User:            &User{ID: 12},
		Account:         &Account{ID: 13},
		RequestID:       "upstream-failed-1",
		Model:           "gemini-2.5-pro",
		UpstreamModel:   "gemini-2.5-pro-preview",
		InboundEndpoint: "/v1/messages",
		HTTPStatus:      502,
		ErrorCode:       "upstream_error",
		ErrorMessage:    "authorization=Bearer-secret-token",
		SimulatedClient: GatewayClientProfileGeminiCLI,
		Stream:          true,
		Duration:        2 * time.Second,
	})
	require.NoError(t, err)
	require.Equal(t, 1, repo.bestEffortCalls)
	require.NotNil(t, repo.lastLog)
	require.Equal(t, UsageLogStatusFailed, repo.lastLog.Status)
	require.NotNil(t, repo.lastLog.ThinkingEnabled)
	require.True(t, *repo.lastLog.ThinkingEnabled)
	require.NotNil(t, repo.lastLog.SimulatedClient)
	require.Equal(t, GatewayClientProfileGeminiCLI, *repo.lastLog.SimulatedClient)
	require.NotNil(t, repo.lastLog.HTTPStatus)
	require.Equal(t, 502, *repo.lastLog.HTTPStatus)
	require.NotNil(t, repo.lastLog.ErrorMessage)
	require.NotContains(t, *repo.lastLog.ErrorMessage, "Bearer-secret-token")
	require.Equal(t, 0.0, repo.lastLog.TotalCost)
	require.Equal(t, 0.0, repo.lastLog.ActualCost)
}
