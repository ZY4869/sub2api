//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatewayServiceRecordUsage_ThinkingEnabledUsesExplicitInputWithoutContext(t *testing.T) {
	usageRepo := &openAIRecordUsageBestEffortLogRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})

	explicit := true
	err := svc.RecordUsage(context.Background(), &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "explicit-thinking-no-context",
			Usage: ClaudeUsage{
				InputTokens:  10,
				OutputTokens: 5,
			},
			Model:    "claude-sonnet-4",
			Duration: time.Second,
		},
		APIKey:          &APIKey{ID: 1},
		User:            &User{ID: 1},
		Account:         &Account{ID: 1},
		ThinkingEnabled: &explicit,
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.ThinkingEnabled)
	require.True(t, *usageRepo.lastLog.ThinkingEnabled)
}

func TestGatewayServiceRecordUsage_ThinkingEnabledExplicitFalseOverridesDetachedContext(t *testing.T) {
	usageRepo := &openAIRecordUsageBestEffortLogRepoStub{}
	svc := newGatewayRecordUsageServiceForTest(usageRepo, &openAIRecordUsageUserRepoStub{}, &openAIRecordUsageSubRepoStub{})

	reqCtx, cancel := context.WithCancel(WithThinkingEnabled(context.Background(), true, false))
	cancel()
	explicit := false

	err := svc.RecordUsage(reqCtx, &RecordUsageInput{
		Result: &ForwardResult{
			RequestID: "explicit-thinking-detached-context",
			Usage: ClaudeUsage{
				InputTokens:  10,
				OutputTokens: 5,
			},
			Model:    "claude-sonnet-4",
			Duration: time.Second,
		},
		APIKey:          &APIKey{ID: 2},
		User:            &User{ID: 2},
		Account:         &Account{ID: 2},
		ThinkingEnabled: &explicit,
	})

	require.NoError(t, err)
	require.NotNil(t, usageRepo.lastLog)
	require.NotNil(t, usageRepo.lastLog.ThinkingEnabled)
	require.False(t, *usageRepo.lastLog.ThinkingEnabled)
}
