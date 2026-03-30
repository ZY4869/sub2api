//go:build unit

package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestGatewayService_SelectAccountForModelWithPlatform_GeminiVertexPrefersGlobalBeforeRegionalLRU(t *testing.T) {
	ctx := context.Background()
	now := time.Now()

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				LastUsedAt:  ptr(now.Add(-30 * time.Minute)),
				Credentials: map[string]any{"oauth_type": "vertex_ai", "vertex_location": "global"},
			},
			{
				ID:          2,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				LastUsedAt:  ptr(now.Add(-2 * time.Hour)),
				Credentials: map[string]any{"oauth_type": "vertex_ai", "vertex_location": "us-central1"},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	svc := &GatewayService{
		accountRepo: repo,
		cache:       &mockGatewayCacheForPlatform{},
		cfg:         testConfig(),
	}

	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "gemini-2.5-pro", nil, PlatformGemini)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(1), account.ID)
}

func TestGatewayService_SelectAccountForModelWithPlatform_GeminiVertexHigherPriorityRegionalStillWins(t *testing.T) {
	ctx := context.Background()

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"oauth_type": "vertex_ai", "vertex_location": "global"},
			},
			{
				ID:          2,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    0,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{"oauth_type": "vertex_ai", "vertex_location": "europe-west4"},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	svc := &GatewayService{
		accountRepo: repo,
		cache:       &mockGatewayCacheForPlatform{},
		cfg:         testConfig(),
	}

	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "gemini-2.5-pro", nil, PlatformGemini)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(2), account.ID)
}

func TestGatewayService_SelectAccountForModelWithPlatform_GeminiPreviewRegionalVertexAllowed(t *testing.T) {
	ctx := context.Background()

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Credentials: map[string]any{
					"oauth_type":      "vertex_ai",
					"vertex_location": "us-central1",
					"model_mapping":   map[string]any{"gemini-3-pro-preview": "gemini-3-pro-preview"},
				},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	svc := &GatewayService{
		accountRepo: repo,
		cache:       &mockGatewayCacheForPlatform{},
		cfg:         testConfig(),
	}

	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "gemini-3-pro-preview", nil, PlatformGemini)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(1), account.ID)
}

func TestGatewayService_SelectAccountForModelWithPlatform_GeminiAPIKeyQuotaExceededSkipsOnlyCurrentAccount(t *testing.T) {
	ctx := context.Background()

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformGemini,
				Type:        AccountTypeAPIKey,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Extra:       map[string]any{"quota_limit": 10.0, "quota_used": 10.0},
			},
			{
				ID:          2,
				Platform:    PlatformGemini,
				Type:        AccountTypeAPIKey,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Extra:       map[string]any{"quota_limit": 10.0, "quota_used": 3.0},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	svc := &GatewayService{
		accountRepo: repo,
		cache:       &mockGatewayCacheForPlatform{},
		cfg:         testConfig(),
	}

	account, err := svc.selectAccountForModelWithPlatform(ctx, nil, "", "gemini-2.5-flash", nil, PlatformGemini)
	require.NoError(t, err)
	require.NotNil(t, account)
	require.Equal(t, int64(2), account.ID)
}

func TestGatewayService_SelectAccountWithLoadAwareness_GeminiVertexPrefersGlobalBeforeRegionalLoad(t *testing.T) {
	ctx := context.Background()
	groupID := int64(901)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:          1,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 5,
				Credentials: map[string]any{"oauth_type": "vertex_ai", "vertex_location": "global"},
			},
			{
				ID:          2,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 5,
				Credentials: map[string]any{"oauth_type": "vertex_ai", "vertex_location": "us-central1"},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Platform: PlatformGemini, Status: StatusActive, Hydrated: true},
		},
	}

	cfg := testConfig()
	cfg.Gateway.Scheduling.LoadBatchEnabled = true

	concurrencyCache := &mockConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			1: {AccountID: 1, LoadRate: 90},
			2: {AccountID: 2, LoadRate: 10},
		},
	}

	svc := &GatewayService{
		accountRepo:        repo,
		groupRepo:          groupRepo,
		cache:              &mockGatewayCacheForPlatform{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	result, err := svc.SelectAccountWithLoadAwareness(ctx, &groupID, "vertex-global", "gemini-2.5-pro", nil, "")
	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(1), result.Account.ID)
}

func TestGatewayService_SelectAccountWithLoadAwareness_GeminiVertexFallsBackToRegionalOnRateLimit(t *testing.T) {
	ctx := context.Background()
	groupID := int64(902)
	rateLimitedUntil := time.Now().Add(10 * time.Minute)

	repo := &mockAccountRepoForPlatform{
		accounts: []Account{
			{
				ID:               1,
				Platform:         PlatformGemini,
				Type:             AccountTypeOAuth,
				Priority:         1,
				Status:           StatusActive,
				Schedulable:      true,
				Concurrency:      5,
				RateLimitResetAt: &rateLimitedUntil,
				Credentials:      map[string]any{"oauth_type": "vertex_ai", "vertex_location": "global"},
			},
			{
				ID:          2,
				Platform:    PlatformGemini,
				Type:        AccountTypeOAuth,
				Priority:    1,
				Status:      StatusActive,
				Schedulable: true,
				Concurrency: 5,
				Credentials: map[string]any{"oauth_type": "vertex_ai", "vertex_location": "asia-east1"},
			},
		},
		accountsByID: map[int64]*Account{},
	}
	for i := range repo.accounts {
		repo.accountsByID[repo.accounts[i].ID] = &repo.accounts[i]
	}

	groupRepo := &mockGroupRepoForGateway{
		groups: map[int64]*Group{
			groupID: {ID: groupID, Platform: PlatformGemini, Status: StatusActive, Hydrated: true},
		},
	}

	cfg := testConfig()
	cfg.Gateway.Scheduling.LoadBatchEnabled = true

	concurrencyCache := &mockConcurrencyCache{
		loadMap: map[int64]*AccountLoadInfo{
			1: {AccountID: 1, LoadRate: 0},
			2: {AccountID: 2, LoadRate: 0},
		},
	}

	svc := &GatewayService{
		accountRepo:        repo,
		groupRepo:          groupRepo,
		cache:              &mockGatewayCacheForPlatform{},
		cfg:                cfg,
		concurrencyService: NewConcurrencyService(concurrencyCache),
	}

	_, regionalFallbackBefore, _ := GeminiVertexRoutingStats()
	result, err := svc.SelectAccountWithLoadAwareness(ctx, &groupID, "vertex-fallback", "gemini-2.5-pro", nil, "")
	_, regionalFallbackAfter, _ := GeminiVertexRoutingStats()

	require.NoError(t, err)
	require.NotNil(t, result)
	require.NotNil(t, result.Account)
	require.Equal(t, int64(2), result.Account.ID)
	require.Equal(t, regionalFallbackBefore+1, regionalFallbackAfter)
}
