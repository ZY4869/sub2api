package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestOpenAIGatewayService_IsModelSupportedByAccount_UsesMappedSourceAndAlias(t *testing.T) {
	svc := &OpenAIGatewayService{}
	account := &Account{
		Platform: PlatformOpenAI,
		Type:     AccountTypeAPIKey,
		Credentials: map[string]any{
			"model_mapping": map[string]any{
				"friendly-gpt": "gpt-4.1-mini",
			},
		},
	}

	require.True(t, svc.isModelSupportedByAccount(account, "friendly-gpt"))
	require.True(t, svc.isModelSupportedByAccount(account, "gpt-4.1-mini"))
	require.False(t, svc.isModelSupportedByAccount(account, "gpt-4o"))
}
