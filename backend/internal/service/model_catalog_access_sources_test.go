//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestCollectModelCatalogAccessSources_UsesDynamicAccountTypes(t *testing.T) {
	record := &modelCatalogRecord{
		model:            "claude-sonnet-4.5",
		canonicalModelID: "claude-sonnet-4-5-20250929",
		provider:         PlatformAnthropic,
		mode:             "chat",
		defaultPlatforms: []string{PlatformAnthropic, PlatformAntigravity},
	}
	accounts := []Account{
		{Platform: PlatformAnthropic, Type: AccountTypeOAuth, Status: StatusActive},
		{Platform: PlatformAnthropic, Type: AccountTypeAPIKey, Status: StatusActive},
		{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive},
	}

	sources := collectModelCatalogAccessSources(context.Background(), &GatewayService{}, record, accounts)
	require.Equal(t, []string{ModelCatalogAccessSourceLogin, ModelCatalogAccessSourceKey}, sources)
}

func TestCollectModelCatalogAccessSources_IgnoresDisabledAccountsAndUnsupportedMappings(t *testing.T) {
	record := &modelCatalogRecord{
		model:            "claude-sonnet-4.5",
		canonicalModelID: "claude-sonnet-4-5-20250929",
		provider:         PlatformAnthropic,
		mode:             "chat",
		defaultPlatforms: []string{PlatformAnthropic, PlatformAntigravity},
	}
	accounts := []Account{
		{Platform: PlatformAnthropic, Type: AccountTypeOAuth, Status: StatusDisabled},
		{
			Platform: PlatformAntigravity,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"gemini-3-flash": "gemini-3-flash",
				},
			},
		},
	}

	sources := collectModelCatalogAccessSources(context.Background(), &GatewayService{}, record, accounts)
	require.Empty(t, sources)
}

func TestCollectModelCatalogAccessSources_SupportCandidatesIncludeHyphenatedAlias(t *testing.T) {
	record := &modelCatalogRecord{
		model:            "claude-sonnet-4.5",
		canonicalModelID: "claude-sonnet-4-5-20250929",
		provider:         PlatformAnthropic,
		mode:             "chat",
		defaultPlatforms: []string{PlatformAntigravity},
	}
	accounts := []Account{
		{
			Platform: PlatformAntigravity,
			Type:     AccountTypeAPIKey,
			Status:   StatusActive,
			Credentials: map[string]any{
				"model_mapping": map[string]any{
					"claude-sonnet-4-5": "claude-sonnet-4-5",
				},
			},
		},
	}

	sources := collectModelCatalogAccessSources(context.Background(), &GatewayService{}, record, accounts)
	require.Equal(t, []string{ModelCatalogAccessSourceKey}, sources)
}

func TestCollectModelCatalogAccessSources_RespectsMixedSchedulingForNativeProtocols(t *testing.T) {
	record := &modelCatalogRecord{
		model:            "claude-sonnet-4.5",
		canonicalModelID: "claude-sonnet-4-5-20250929",
		provider:         PlatformAnthropic,
		mode:             "chat",
	}
	accounts := []Account{{
		Platform: PlatformAntigravity,
		Type:     AccountTypeUpstream,
		Status:   StatusActive,
		Extra: map[string]any{
			"mixed_scheduling": false,
		},
	}}

	sources := collectModelCatalogAccessSources(context.Background(), &GatewayService{}, record, accounts)
	require.Empty(t, sources)

	accounts[0].Extra["mixed_scheduling"] = true
	sources = collectModelCatalogAccessSources(context.Background(), &GatewayService{}, record, accounts)
	require.Equal(t, []string{ModelCatalogAccessSourceKey}, sources)
}

func TestCollectModelCatalogAccessSources_SoraUsesSoraPlatformInsteadOfOpenAI(t *testing.T) {
	record := &modelCatalogRecord{
		model:            "sora2",
		canonicalModelID: "sora2",
		provider:         PlatformOpenAI,
		mode:             "video",
		defaultPlatforms: []string{PlatformSora},
	}
	accounts := []Account{
		{Platform: PlatformOpenAI, Type: AccountTypeAPIKey, Status: StatusActive},
		{Platform: PlatformSora, Type: AccountTypeSetupToken, Status: StatusActive},
	}

	sources := collectModelCatalogAccessSources(context.Background(), &GatewayService{}, record, accounts)
	require.Equal(t, []string{ModelCatalogAccessSourceLogin}, sources)
}
