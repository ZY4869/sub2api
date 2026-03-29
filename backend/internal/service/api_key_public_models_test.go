//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGatewayService_ResolveAPIKeySelectionModel_SourceOnlyUsesAlias(t *testing.T) {
	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Name:        "anthropic-apikey",
				Platform:    PlatformAnthropic,
				Type:        AccountTypeAPIKey,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"model_mapping": map[string]any{
						"friendly-sonnet": "claude-sonnet-4-20250514",
					},
				},
			},
		},
	}
	svc := &GatewayService{accountRepo: repo}
	apiKey := &APIKey{
		ID:               10,
		ModelDisplayMode: APIKeyModelDisplayModeSourceOnly,
		GroupBindings: []APIKeyGroupBinding{
			{
				GroupID: 20,
				Group: &Group{
					ID:       20,
					Name:     "anthropic-group",
					Platform: PlatformAnthropic,
					Status:   StatusActive,
				},
			},
		},
	}

	entry, ok := svc.FindAPIKeyPublicModel(context.Background(), apiKey, PlatformAnthropic, "claude-sonnet-4-20250514")
	require.True(t, ok)
	require.Equal(t, "friendly-sonnet", entry.AliasID)
	require.Equal(t, "claude-sonnet-4-20250514", entry.PublicID)
	require.Equal(t, "claude-sonnet-4-20250514", entry.SourceID)

	got := svc.ResolveAPIKeySelectionModel(context.Background(), apiKey, PlatformAnthropic, "claude-sonnet-4-20250514")
	require.Equal(t, "friendly-sonnet", got)
}
