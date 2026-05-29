package service

import (
	"context"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

type CRSSyncService struct {
	accountRepo        AccountRepository
	proxyRepo          ProxyRepository
	oauthService       *OAuthService
	openaiOAuthService *OpenAIOAuthService
	geminiOAuthService *GeminiOAuthService
	cfg                *config.Config
}

func NewCRSSyncService(
	accountRepo AccountRepository,
	proxyRepo ProxyRepository,
	oauthService *OAuthService,
	openaiOAuthService *OpenAIOAuthService,
	geminiOAuthService *GeminiOAuthService,
	cfg *config.Config,
) *CRSSyncService {
	return &CRSSyncService{
		accountRepo:        accountRepo,
		proxyRepo:          proxyRepo,
		oauthService:       oauthService,
		openaiOAuthService: openaiOAuthService,
		geminiOAuthService: geminiOAuthService,
		cfg:                cfg,
	}
}

func (s *CRSSyncService) SyncFromCRS(ctx context.Context, input SyncFromCRSInput) (*SyncFromCRSResult, error) {
	exported, err := s.fetchCRSExport(ctx, input.BaseURL, input.Username, input.Password)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC().Format(time.RFC3339)

	result := &SyncFromCRSResult{
		Items: make(
			[]SyncFromCRSItemResult,
			0,
			len(exported.Data.ClaudeAccounts)+len(exported.Data.ClaudeConsoleAccounts)+len(exported.Data.OpenAIOAuthAccounts)+len(exported.Data.OpenAIResponsesAccounts)+len(exported.Data.GeminiOAuthAccounts)+len(exported.Data.GeminiAPIKeyAccounts),
		),
	}

	selectedSet := buildSelectedSet(input.SelectedAccountIDs)

	var proxies []Proxy
	if input.SyncProxies {
		proxies, _ = s.proxyRepo.ListActive(ctx)
	}

	s.syncCRSClaudeAccounts(ctx, input, exported, result, selectedSet, &proxies, now)

	s.syncCRSClaudeConsoleAccounts(ctx, input, exported, result, selectedSet, &proxies, now)

	s.syncCRSOpenAIOAuthAccounts(ctx, input, exported, result, selectedSet, &proxies, now)

	s.syncCRSOpenAIResponsesAccounts(ctx, input, exported, result, selectedSet, &proxies, now)

	s.syncCRSGeminiOAuthAccounts(ctx, input, exported, result, selectedSet, &proxies, now)

	s.syncCRSGeminiAPIKeyAccounts(ctx, input, exported, result, selectedSet, &proxies, now)

	return result, nil
}
