package service

import (
	"context"
	"fmt"
	"math/rand/v2"
	"time"
)

func (s *AccountUsageService) getAnthropicActiveUsage(ctx context.Context, account *Account, force bool) (*UsageInfo, error) {
	apiResp, passiveUsage, err := s.getCachedOAuthUsageRaw(ctx, account, force)
	if err != nil {
		return nil, err
	}
	if passiveUsage != nil {
		return passiveUsage, nil
	}

	// 构建 UsageInfo（每次都重新计算 RemainingSeconds）
	now := time.Now()
	usage := s.buildUsageInfo(apiResp, &now)

	// 添加窗口统计（有独立缓存，1 分钟）
	s.addWindowStats(ctx, account, usage, force)

	// 将主动查询结果同步到被动缓存，下次 passive 加载即为最新值
	s.syncActiveToPassive(ctx, account.ID, usage)

	s.tryClearRecoverableAccountError(ctx, account)
	return usage, nil
}

func (s *AccountUsageService) getCachedOAuthUsageRaw(ctx context.Context, account *Account, force bool) (*ClaudeUsageResponse, *UsageInfo, error) {
	if account == nil {
		return nil, nil, fmt.Errorf("account is nil")
	}
	accountID := account.ID
	var apiResp *ClaudeUsageResponse

	if !force {
		cachedResp, cachedErr := s.loadOAuthUsageCache(accountID)
		if cachedErr != nil {
			if passiveUsage, ok := s.passiveUsageFallbackForMissingAccessToken(ctx, account, cachedErr); ok {
				return nil, passiveUsage, nil
			}
			return nil, nil, cachedErr
		}
		apiResp = cachedResp
	}

	if apiResp != nil {
		return apiResp, nil, nil
	}

	// 随机延迟：打散多账号并发请求，避免同一时刻大量相同 TLS 指纹请求
	// 触发上游反滥用检测。延迟范围 0~800ms，仅在缓存未命中时生效。
	jitter := time.Duration(rand.Int64N(int64(apiQueryMaxJitter)))
	select {
	case <-time.After(jitter):
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	}

	flightKey := fmt.Sprintf("usage:%d", accountID)
	if force {
		flightKey = fmt.Sprintf("usage:%d:force", accountID)
	}
	result, flightErr, _ := s.cache.apiFlight.Do(flightKey, func() (any, error) {
		if !force {
			if cachedResp, cachedErr := s.loadOAuthUsageCache(accountID); cachedResp != nil || cachedErr != nil {
				return cachedResp, cachedErr
			}
		}
		resp, fetchErr := s.fetchOAuthUsageRaw(ctx, account)
		if fetchErr != nil {
			s.cache.apiCache.Store(accountID, &apiUsageCache{
				err:       fetchErr,
				timestamp: time.Now(),
			})
			return nil, fetchErr
		}
		s.cache.apiCache.Store(accountID, &apiUsageCache{
			response:  resp,
			timestamp: time.Now(),
		})
		return resp, nil
	})
	if flightErr != nil {
		if passiveUsage, ok := s.passiveUsageFallbackForMissingAccessToken(ctx, account, flightErr); ok {
			return nil, passiveUsage, nil
		}
		return nil, nil, flightErr
	}
	apiResp, _ = result.(*ClaudeUsageResponse)
	return apiResp, nil, nil
}

func (s *AccountUsageService) loadOAuthUsageCache(accountID int64) (*ClaudeUsageResponse, error) {
	if s == nil || s.cache == nil {
		return nil, nil
	}
	cached, ok := s.cache.apiCache.Load(accountID)
	if !ok {
		return nil, nil
	}
	cache, ok := cached.(*apiUsageCache)
	if !ok {
		return nil, nil
	}
	age := time.Since(cache.timestamp)
	if cache.err != nil && age < apiErrorCacheTTL {
		return nil, cache.err
	}
	if cache.response != nil && age < apiCacheTTL {
		return cache.response, nil
	}
	return nil, nil
}

// fetchOAuthUsageRaw 从 Anthropic API 获取原始响应（不构建 UsageInfo）
// 如果账号开启了 TLS 指纹，则使用 TLS 指纹伪装
// 如果有缓存的 Fingerprint，则使用缓存的 User-Agent 等信息
func (s *AccountUsageService) fetchOAuthUsageRaw(ctx context.Context, account *Account) (*ClaudeUsageResponse, error) {
	accessToken := account.GetCredential("access_token")
	if accessToken == "" {
		return nil, fmt.Errorf("no access token available")
	}

	var proxyURL string
	if account.ProxyID != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}

	// 构建完整的选项
	opts := &ClaudeUsageFetchOptions{
		AccessToken: accessToken,
		ProxyURL:    proxyURL,
		AccountID:   account.ID,
		TLSProfile:  resolveAccountTLSFingerprintProfile(account, s.tlsFingerprintProfileService),
	}

	// 尝试获取缓存的 Fingerprint（包含 User-Agent 等信息）
	if s.identityCache != nil {
		if fp, err := s.identityCache.GetFingerprint(ctx, account.ID); err == nil && fp != nil {
			opts.Fingerprint = fp
		}
	}

	return s.usageFetcher.FetchUsageWithOptions(ctx, opts)
}
