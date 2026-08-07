package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestIsGeminiNonBillablePassthroughEndpoint_StrictOfficialSecondStageSurfaces(t *testing.T) {
	tests := []string{
		"/v1beta/corpora",
		"/v1beta/corpora/corpus-1/operations/op-1",
		"/v1beta/corpora/corpus-1/permissions/perm-1",
		"/v1beta/dynamic/dynamic-1:generateContent",
		"/v1beta/generatedFiles",
		"/v1beta/generatedFiles/generated-file-1/operations/op-1",
		"/v1beta/models/gemini-2.5-pro/operations",
		"/v1beta/tunedModels",
		"/v1beta/tunedModels/tuned-model-1:asyncBatchEmbedContent",
		"/v1beta/tunedModels/tuned-model-1/permissions/perm-1",
		"/v1beta/tunedModels/tuned-model-1/operations/op-1",
	}

	for _, path := range tests {
		t.Run(path, func(t *testing.T) {
			require.True(t, isGeminiNonBillablePassthroughEndpoint(path))
			require.False(t, isGeminiBillingEndpoint(path))
		})
	}
}

func TestGatewayServiceRecordUsage_BillingErrorRetainsFailedZeroChargeUsageLog(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{}
	billingRepo := &openAIRecordUsageBillingRepoStub{err: errors.New("billing tx failed x-api-key=secret-value")}
	cfg := &config.Config{}
	cfg.Default.RateMultiplier = 1.0
	svc := &GatewayService{
		cfg:              cfg,
		usageLogRepo:     usageRepo,
		usageBillingRepo: billingRepo,
		userRepo:         &openAIRecordUsageUserRepoStub{},
		userSubRepo:      &openAIRecordUsageSubRepoStub{},
		billingService:   NewBillingService(cfg, nil),
		deferredService:  &DeferredService{},
	}

	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "anthropic_billing_fail",
			Usage: ClaudeUsage{
				InputTokens:  8,
				OutputTokens: 4,
			},
			Model:    "claude-3-5-sonnet-20241022",
			Stream:   true,
			Duration: time.Second,
		},
		APIKey:  &APIKey{ID: 6101},
		User:    &User{ID: 6201},
		Account: &Account{ID: 6301, Platform: PlatformAnthropic, Type: AccountTypeOAuth},
	})

	require.Error(t, err)
	require.Equal(t, 1, billingRepo.calls)
	require.Equal(t, 1, usageRepo.calls)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, UsageLogStatusFailed, usageRepo.lastLog.Status)
	require.Equal(t, "anthropic_billing_fail", usageRepo.lastLog.RequestID)
	require.Equal(t, RequestTypeStream, usageRepo.lastLog.EffectiveRequestType())
	require.Greater(t, usageRepo.lastLog.TotalCost, 0.0)
	require.Equal(t, 0.0, usageRepo.lastLog.ActualCost)
	require.Equal(t, 0.0, usageRepo.lastLog.ActualCostUSDEquivalent)
	require.Nil(t, usageRepo.lastLog.ActualCostByCurrency)
	require.NotNil(t, usageRepo.lastLog.ErrorCode)
	require.Equal(t, usageBillingFailureErrorCode, *usageRepo.lastLog.ErrorCode)
	require.NotNil(t, usageRepo.lastLog.ErrorMessage)
	require.NotContains(t, *usageRepo.lastLog.ErrorMessage, "secret-value")
}
