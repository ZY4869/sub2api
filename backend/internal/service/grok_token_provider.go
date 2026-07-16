package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

const (
	grokTokenRefreshSkew        = 3 * time.Minute
	grokTokenCacheSkew          = 5 * time.Minute
	grokRequestRefreshTimeout   = 8 * time.Second
	grokRefreshLockWaitTimeout  = 2 * time.Second
	grokRefreshLockPollInterval = 25 * time.Millisecond
)

var (
	errGrokOAuthRefreshNotConfigured = errors.New("grok oauth refresh is not configured")
	errGrokOAuthRefreshTokenMissing  = errors.New("grok oauth refresh token is missing")
	errGrokOAuthAccessTokenMissing   = errors.New("grok oauth access token is missing")
	errGrokOAuthAccessTokenExpired   = errors.New("grok oauth access token is expired")
)

type GrokTokenCache = GeminiTokenCache

type GrokTokenProvider struct {
	accountRepo   AccountRepository
	tokenCache    GrokTokenCache
	refreshAPI    *OAuthRefreshAPI
	executor      OAuthRefreshExecutor
	refreshPolicy ProviderRefreshPolicy
}

func NewGrokTokenProvider(accountRepo AccountRepository, tokenCache GrokTokenCache) *GrokTokenProvider {
	return &GrokTokenProvider{
		accountRepo:   accountRepo,
		tokenCache:    tokenCache,
		refreshPolicy: GrokProviderRefreshPolicy(),
	}
}

func (p *GrokTokenProvider) SetRefreshAPI(api *OAuthRefreshAPI, executor OAuthRefreshExecutor) {
	p.refreshAPI = api
	p.executor = executor
}

func (p *GrokTokenProvider) SetRefreshPolicy(policy ProviderRefreshPolicy) {
	p.refreshPolicy = policy
}

func (p *GrokTokenProvider) GetAccessToken(ctx context.Context, account *Account) (string, error) {
	if account == nil {
		return "", errors.New("account is nil")
	}
	if !account.IsGrokOAuth() {
		return "", errors.New("not a grok oauth account")
	}
	if strings.TrimSpace(account.GetCredential("refresh_token")) == "" {
		return "", errGrokOAuthRefreshTokenMissing
	}

	cacheKey := GrokTokenCacheKey(account)
	accountAccessToken := strings.TrimSpace(account.GetGrokOAuthAccessToken())
	expiresAt := account.GetCredentialAsTime("expires_at")
	if p.tokenCache != nil && accountAccessToken != "" && expiresAt != nil && time.Until(*expiresAt) > grokTokenRefreshSkew {
		if token, err := p.tokenCache.GetAccessToken(ctx, cacheKey); err == nil && strings.TrimSpace(token) == accountAccessToken {
			return accountAccessToken, nil
		}
	}

	needsRefresh := accountAccessToken == "" || expiresAt == nil || time.Until(*expiresAt) <= grokTokenRefreshSkew
	if needsRefresh {
		if p.refreshAPI == nil || p.executor == nil {
			return "", errGrokOAuthRefreshNotConfigured
		}
		refreshCtx, cancel := context.WithTimeout(ctx, grokRequestRefreshTimeout)
		defer cancel()
		result, err := p.refreshAPI.RefreshIfNeeded(refreshCtx, account, p.executor, grokTokenRefreshSkew)
		if err != nil {
			if p.refreshPolicy.OnRefreshError == ProviderRefreshErrorReturn {
				return "", err
			}
			slog.Warn("grok_token_refresh_failed_use_existing", "account_id", account.ID, "error", err)
		} else if result != nil && result.LockHeld {
			if p.refreshPolicy.OnLockHeld == ProviderLockHeldWaitForCache {
				if token, waitErr := p.waitForRefreshedToken(refreshCtx, account, cacheKey); waitErr != nil || strings.TrimSpace(token) != "" {
					return token, waitErr
				}
			}
		} else if result != nil && result.Account != nil {
			account = result.Account
			expiresAt = account.GetCredentialAsTime("expires_at")
		}
	}

	accessToken := strings.TrimSpace(account.GetGrokOAuthAccessToken())
	if accessToken == "" {
		return "", errGrokOAuthAccessTokenMissing
	}
	if expiresAt != nil && !time.Now().Before(*expiresAt) {
		return "", errGrokOAuthAccessTokenExpired
	}

	if p.tokenCache != nil {
		latestAccount, isStale := CheckTokenVersion(ctx, account, p.accountRepo)
		if isStale && latestAccount != nil {
			accessToken = strings.TrimSpace(latestAccount.GetGrokOAuthAccessToken())
			if accessToken == "" {
				return "", errGrokOAuthAccessTokenMissing
			}
			expiresAt = latestAccount.GetCredentialAsTime("expires_at")
			if expiresAt != nil && !time.Now().Before(*expiresAt) {
				return "", errGrokOAuthAccessTokenExpired
			}
		}
		ttl := 30 * time.Minute
		if expiresAt != nil {
			until := time.Until(*expiresAt)
			switch {
			case until > grokTokenCacheSkew:
				ttl = until - grokTokenCacheSkew
			case until > 0:
				ttl = until
			default:
				ttl = time.Minute
			}
		}
		if err := p.tokenCache.SetAccessToken(ctx, cacheKey, accessToken, ttl); err != nil {
			slog.Warn("grok_token_cache_set_failed", "account_id", account.ID, "error", err)
		}
	}

	return accessToken, nil
}

func (p *GrokTokenProvider) waitForRefreshedToken(ctx context.Context, account *Account, cacheKey string) (string, error) {
	waitCtx, cancel := context.WithTimeout(ctx, grokRefreshLockWaitTimeout)
	defer cancel()

	initialToken := strings.TrimSpace(account.GetGrokOAuthAccessToken())
	initialVersion := account.GetCredentialAsInt64("_token_version")
	ticker := time.NewTicker(grokRefreshLockPollInterval)
	defer ticker.Stop()
	for {
		if p.accountRepo != nil {
			latest, err := p.accountRepo.GetByID(waitCtx, account.ID)
			if err == nil && latest != nil {
				token := strings.TrimSpace(latest.GetGrokOAuthAccessToken())
				version := latest.GetCredentialAsInt64("_token_version")
				expiresAt := latest.GetCredentialAsTime("expires_at")
				if token != "" && token != initialToken && version >= initialVersion && expiresAt != nil && time.Now().Before(*expiresAt) {
					if p.tokenCache != nil {
						ttl := time.Until(*expiresAt)
						if ttl > grokTokenCacheSkew {
							ttl -= grokTokenCacheSkew
						}
						_ = p.tokenCache.SetAccessToken(waitCtx, cacheKey, token, ttl)
					}
					return token, nil
				}
			}
		}
		if p.tokenCache != nil {
			if token, err := p.tokenCache.GetAccessToken(waitCtx, cacheKey); err == nil {
				if token = strings.TrimSpace(token); token != "" && token != initialToken {
					return token, nil
				}
			}
		}
		select {
		case <-waitCtx.Done():
			return "", fmt.Errorf("grok token refresh still in progress")
		case <-ticker.C:
		}
	}
}
