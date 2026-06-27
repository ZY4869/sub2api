package service

import (
	"context"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func openAICapabilityTestAccount(id int64, capabilities []string, priority int) Account {
	credentials := map[string]any{"api_key": "sk-test"}
	if capabilities != nil {
		credentials[openAIEndpointCapabilitiesCredentialKey] = capabilities
	}
	return Account{
		ID:          id,
		Platform:    PlatformOpenAI,
		Type:        AccountTypeAPIKey,
		Status:      StatusActive,
		Schedulable: true,
		Concurrency: 1,
		Priority:    priority,
		GroupIDs:    []int64{9901},
		Credentials: credentials,
	}
}

func unsupportedEmbeddingsAccount(id int64, priority int) Account {
	return openAICapabilityTestAccount(id, []string{string(OpenAIEndpointCapabilityChatCompletions)}, priority)
}

func supportedEmbeddingsAccount(id int64, priority int) Account {
	return openAICapabilityTestAccount(id, nil, priority)
}

func releaseOpenAISelection(t *testing.T, selection *AccountSelectionResult) {
	t.Helper()
	if selection != nil && selection.ReleaseFunc != nil {
		selection.ReleaseFunc()
	}
}

func TestOpenAIGatewayService_SelectAccountWithSchedulerForCapability_SkipsUnsupportedStickyEmbeddingsAccount(t *testing.T) {
	groupID := int64(9901)
	unsupported := unsupportedEmbeddingsAccount(50101, 0)
	supported := supportedEmbeddingsAccount(50102, 5)
	cache := &stubGatewayCache{sessionBindings: map[string]int64{"openai:session_embed_cap": unsupported.ID}}
	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{unsupported, supported}},
		cache:              cache,
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithSchedulerForCapability(
		context.Background(),
		&groupID,
		"",
		"session_embed_cap",
		"text-embedding-3-small",
		nil,
		OpenAIUpstreamTransportHTTPSSE,
		OpenAIEndpointCapabilityEmbeddings,
	)

	require.NoError(t, err)
	defer releaseOpenAISelection(t, selection)
	require.Equal(t, supported.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickySessionHit)
	require.Equal(t, 1, decision.CandidateCount)
	require.Equal(t, 1, cache.deletedSessions["openai:session_embed_cap"])
}

func TestOpenAIGatewayService_SelectAccountWithSchedulerForCapability_SkipsUnsupportedPreviousResponseEmbeddingsAccount(t *testing.T) {
	groupID := int64(9901)
	unsupported := unsupportedEmbeddingsAccount(50201, 0)
	unsupported.Extra = map[string]any{"openai_apikey_responses_websockets_v2_enabled": true}
	supported := supportedEmbeddingsAccount(50202, 5)
	cfg := newOpenAIWSV2TestConfig()
	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{unsupported, supported}},
		cache:              &stubGatewayCache{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}
	store := svc.getOpenAIWSStateStore()
	require.NoError(t, store.BindResponseAccount(context.Background(), groupID, "resp_embed_cap", unsupported.ID, time.Hour))

	selection, decision, err := svc.SelectAccountWithSchedulerForCapability(
		context.Background(),
		&groupID,
		"resp_embed_cap",
		"",
		"text-embedding-3-small",
		nil,
		OpenAIUpstreamTransportHTTPSSE,
		OpenAIEndpointCapabilityEmbeddings,
	)

	require.NoError(t, err)
	defer releaseOpenAISelection(t, selection)
	require.Equal(t, supported.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.False(t, decision.StickyPreviousHit)
	require.Equal(t, 1, decision.CandidateCount)
}

func TestOpenAIGatewayService_SelectAccountWithSchedulerForCapability_LoadBalanceSkipsUnsupportedEmbeddingsAccount(t *testing.T) {
	groupID := int64(9901)
	unsupported := unsupportedEmbeddingsAccount(50301, 0)
	supported := supportedEmbeddingsAccount(50302, 9)
	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{unsupported, supported}},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}

	selection, decision, err := svc.SelectAccountWithSchedulerForCapability(
		context.Background(),
		&groupID,
		"",
		"",
		"text-embedding-3-small",
		nil,
		OpenAIUpstreamTransportHTTPSSE,
		OpenAIEndpointCapabilityEmbeddings,
	)

	require.NoError(t, err)
	defer releaseOpenAISelection(t, selection)
	require.Equal(t, supported.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 1, decision.CandidateCount)
}

func TestOpenAIGatewayService_SelectAccountWithSchedulerForCapability_PinnedCatalogAccountStillRequiresEmbeddingsCapability(t *testing.T) {
	groupID := int64(9901)
	unsupported := unsupportedEmbeddingsAccount(50401, 0)
	supported := supportedEmbeddingsAccount(50402, 5)
	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{unsupported, supported}},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}
	ctx := AttachPublishedPublicCatalogEntry(context.Background(), &PublishedPublicCatalogEntry{
		EntryID:         "entry_embed_cap",
		PublicModelID:   "embed-public",
		SourceAccountID: unsupported.ID,
		SourceModelID:   "text-embedding-3-small",
		SourceProtocol:  PlatformOpenAI,
	})

	selection, decision, err := svc.SelectAccountWithSchedulerForCapability(
		ctx,
		&groupID,
		"",
		"",
		"embed-public",
		nil,
		OpenAIUpstreamTransportHTTPSSE,
		OpenAIEndpointCapabilityEmbeddings,
	)

	require.NoError(t, err)
	defer releaseOpenAISelection(t, selection)
	require.Equal(t, supported.ID, selection.Account.ID)
	require.Equal(t, openAIAccountScheduleLayerLoadBalance, decision.Layer)
	require.Equal(t, 1, decision.CandidateCount)
}

func TestOpenAIGatewayService_SelectAccountWithScheduler_ModelUnsupportedReturnsModelNotFound(t *testing.T) {
	groupID := int64(9901)
	unsupported := openAICapabilityTestAccount(50501, nil, 0)
	unsupported.Extra = map[string]any{
		"model_scope_v2": map[string]any{
			"policy_mode": AccountModelPolicyModeWhitelist,
			"entries": []any{
				map[string]any{
					"display_model_id": "gpt-5.4",
					"target_model_id":  "gpt-5.4",
				},
			},
		},
	}
	svc := &OpenAIGatewayService{
		accountRepo:        stubOpenAIAccountRepo{accounts: []Account{unsupported}},
		cfg:                &config.Config{},
		concurrencyService: NewConcurrencyService(stubConcurrencyCache{}),
	}

	selection, _, err := svc.SelectAccountWithScheduler(
		context.Background(),
		&groupID,
		"",
		"",
		"gpt-5.5",
		nil,
		OpenAIUpstreamTransportAny,
	)

	require.ErrorIs(t, err, ErrOpenAIModelNotFound)
	require.Nil(t, selection)
}
