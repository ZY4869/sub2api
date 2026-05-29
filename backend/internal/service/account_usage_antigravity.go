package service

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"
)

// getAntigravityUsage 获取 Antigravity 账户额度
func (s *AccountUsageService) getAntigravityUsage(ctx context.Context, account *Account, force bool) (*UsageInfo, error) {
	if s.antigravityQuotaFetcher == nil || !s.antigravityQuotaFetcher.CanFetch(account) {
		now := time.Now()
		return &UsageInfo{UpdatedAt: &now}, nil
	}

	if !force {

		// 1. 检查缓存
		if cached, ok := s.cache.antigravityCache.Load(account.ID); ok {
			if cache, ok := cached.(*antigravityUsageCache); ok {
				ttl := antigravityCacheTTL(cache.usageInfo)
				if time.Since(cache.timestamp) < ttl {
					usage := cache.usageInfo
					if usage.FiveHour != nil && usage.FiveHour.ResetsAt != nil {
						usage.FiveHour.RemainingSeconds = int(time.Until(*usage.FiveHour.ResetsAt).Seconds())
					}
					return usage, nil
				}
			}
		}

		// 2. singleflight 防止并发击穿
	}

	flightKey := fmt.Sprintf("ag-usage:%d", account.ID)
	if force {
		flightKey = fmt.Sprintf("ag-usage:%d:force", account.ID)
	}
	result, flightErr, _ := s.cache.antigravityFlight.Do(flightKey, func() (any, error) {
		if !force {
			// 再次检查缓存（等待期间可能已被填充）
			if cached, ok := s.cache.antigravityCache.Load(account.ID); ok {
				if cache, ok := cached.(*antigravityUsageCache); ok {
					ttl := antigravityCacheTTL(cache.usageInfo)
					if time.Since(cache.timestamp) < ttl {
						usage := cache.usageInfo
						// 重新计算 RemainingSeconds，避免返回过时的剩余秒数
						recalcAntigravityRemainingSeconds(usage)
						return usage, nil
					}
				}
			}
		}

		fetchParentCtx := ctx
		if fetchParentCtx == nil {
			fetchParentCtx = context.Background()
		}
		fetchCtx, fetchCancel := context.WithTimeout(fetchParentCtx, 30*time.Second)
		defer fetchCancel()

		proxyURL := s.antigravityQuotaFetcher.GetProxyURL(fetchCtx, account)
		fetchResult, err := s.antigravityQuotaFetcher.FetchQuota(fetchCtx, account, proxyURL)
		if err != nil {
			degraded := buildAntigravityDegradedUsage(err)
			enrichUsageWithAccountError(degraded, account)
			s.cache.antigravityCache.Store(account.ID, &antigravityUsageCache{
				usageInfo: degraded,
				timestamp: time.Now(),
			})
			return degraded, nil
		}

		enrichUsageWithAccountError(fetchResult.UsageInfo, account)
		s.cache.antigravityCache.Store(account.ID, &antigravityUsageCache{
			usageInfo: fetchResult.UsageInfo,
			timestamp: time.Now(),
		})
		return fetchResult.UsageInfo, nil
	})

	if flightErr != nil {
		return nil, flightErr
	}
	usage, ok := result.(*UsageInfo)
	if !ok || usage == nil {
		now := time.Now()
		return &UsageInfo{UpdatedAt: &now}, nil
	}
	return usage, nil
}

// recalcAntigravityRemainingSeconds 重新计算 Antigravity UsageInfo 中各窗口的 RemainingSeconds
// 用于从缓存取出时更新倒计时，避免返回过时的剩余秒数
func recalcAntigravityRemainingSeconds(info *UsageInfo) {
	if info == nil {
		return
	}
	if info.FiveHour != nil && info.FiveHour.ResetsAt != nil {
		remaining := int(time.Until(*info.FiveHour.ResetsAt).Seconds())
		if remaining < 0 {
			remaining = 0
		}
		info.FiveHour.RemainingSeconds = remaining
	}
}

// antigravityCacheTTL 根据 UsageInfo 内容决定缓存 TTL
// 403 forbidden 状态稳定，缓存与成功相同（3 分钟）；
// 其他错误（401/网络）可能快速恢复，缓存 1 分钟。
func antigravityCacheTTL(info *UsageInfo) time.Duration {
	if info == nil {
		return antigravityErrorTTL
	}
	if info.IsForbidden {
		return apiCacheTTL // 封号/验证状态不会很快变
	}
	if info.ErrorCode != "" || info.Error != "" {
		return antigravityErrorTTL
	}
	return apiCacheTTL
}

// buildAntigravityDegradedUsage 从 FetchQuota 错误构建降级 UsageInfo
func buildAntigravityDegradedUsage(err error) *UsageInfo {
	now := time.Now()
	errMsg := fmt.Sprintf("usage API error: %v", err)
	slog.Warn("antigravity usage fetch failed, returning degraded response", "error", err)

	info := &UsageInfo{
		UpdatedAt: &now,
		Error:     errMsg,
	}

	// 从错误信息推断 error_code 和状态标记
	// 错误格式来自 antigravity/client.go: "fetchAvailableModels 失败 (HTTP %d): ..."
	errStr := err.Error()
	switch {
	case strings.Contains(errStr, "HTTP 401") ||
		strings.Contains(errStr, "UNAUTHENTICATED") ||
		strings.Contains(errStr, "invalid_grant"):
		info.ErrorCode = errorCodeUnauthenticated
		info.NeedsReauth = true
	case strings.Contains(errStr, "HTTP 429"):
		info.ErrorCode = errorCodeRateLimited
	default:
		info.ErrorCode = errorCodeNetworkError
	}

	return info
}

// enrichUsageWithAccountError 结合账号错误状态修正 UsageInfo
// 场景 1（成功路径）：FetchAvailableModels 正常返回，但账号已因 403 被标记为 error，
//
//	需要在正常 usage 数据上附加 forbidden/validation 信息。
//
// 场景 2（降级路径）：被封号的账号 OAuth token 失效，FetchAvailableModels 返回 401，
//
//	降级逻辑设置了 needs_reauth，但账号实际是 403 封号/需验证，需覆盖为正确状态。
func enrichUsageWithAccountError(info *UsageInfo, account *Account) {
	if info == nil || account == nil || account.Status != StatusError {
		return
	}
	msg := strings.ToLower(account.ErrorMessage)
	if !strings.Contains(msg, "403") && !strings.Contains(msg, "forbidden") &&
		!strings.Contains(msg, "violation") && !strings.Contains(msg, "validation") {
		return
	}
	fbType := classifyForbiddenType(account.ErrorMessage)
	info.IsForbidden = true
	info.ForbiddenType = fbType
	info.ForbiddenReason = account.ErrorMessage
	info.NeedsVerify = fbType == forbiddenTypeValidation
	info.IsBanned = fbType == forbiddenTypeViolation
	info.ValidationURL = extractValidationURL(account.ErrorMessage)
	info.ErrorCode = errorCodeForbidden
	info.NeedsReauth = false
}
