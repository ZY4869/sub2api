package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAccountIsSchedulableForModelWithContext_OpenAIPro_ScopeRateLimitsDoNotMix(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	resetAt := now.Add(30 * time.Minute).UTC().Format(time.RFC3339)

	account := &Account{
		ID:          1,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				openAICodexScopeSpark: map[string]any{
					"rate_limit_reset_at": resetAt,
				},
			},
		},
	}

	require.False(t, account.IsSchedulableForModelWithContext(context.Background(), openAICodexScopeSpark))
	require.True(t, account.IsSchedulableForModelWithContext(context.Background(), openAICodexScopeNormal))
}

func TestAccountIsSchedulableForModelWithContext_OpenAIPro_NormalRateLimitDoesNotAffectSpark(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	resetAt := now.Add(30 * time.Minute).UTC().Format(time.RFC3339)

	account := &Account{
		ID:          2,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeOAuth,
		Status:      StatusActive,
		Schedulable: true,
		Credentials: map[string]any{
			"plan_type": "pro",
		},
		Extra: map[string]any{
			"model_rate_limits": map[string]any{
				openAICodexScopeNormal: map[string]any{
					"rate_limit_reset_at": resetAt,
				},
			},
		},
	}

	require.False(t, account.IsSchedulableForModelWithContext(context.Background(), openAICodexScopeNormal))
	require.True(t, account.IsSchedulableForModelWithContext(context.Background(), openAICodexScopeSpark))
}

