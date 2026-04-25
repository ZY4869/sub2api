package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayServiceRecordUsage_BillsMappedRequestsUsingRequestedModel(t *testing.T) {
	usageRepo := &openAIRecordUsageLogRepoStub{inserted: true}
	userRepo := &openAIRecordUsageUserRepoStub{}
	subRepo := &openAIRecordUsageSubRepoStub{}
	svc := newOpenAIRecordUsageServiceForTest(usageRepo, userRepo, subRepo, nil)
	usage := OpenAIUsage{InputTokens: 20, OutputTokens: 10}

	expectedCost, err := svc.billingService.CalculateCost("gpt-5.4", UsageTokens{
		InputTokens:  20,
		OutputTokens: 10,
	}, 1.1)
	require.NoError(t, err)

	err = svc.RecordUsage(context.Background(), &OpenAIRecordUsageInput{
		Result: &OpenAIForwardResult{
			RequestID:     "resp_upstream_model_billing_fallback",
			Model:         "gpt-5.4",
			UpstreamModel: "gpt-5.3-codex-spark",
			Usage:         usage,
			Duration:      time.Second,
		},
		APIKey:  &APIKey{ID: 10},
		User:    &User{ID: 20},
		Account: &Account{ID: 30},
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.Equal(t, "gpt-5.4", usageRepo.lastLog.Model)
	require.Equal(t, expectedCost.ActualCost, usageRepo.lastLog.ActualCost)
	require.Equal(t, expectedCost.TotalCost, usageRepo.lastLog.TotalCost)
	require.Equal(t, expectedCost.ActualCost, userRepo.lastAmount)
}
