package service

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *RateLimitService) HandleUpstreamError(ctx context.Context, account *Account, statusCode int, headers http.Header, responseBody []byte) (shouldDisable bool) {
	if match := DetectHardBannedAccount(statusCode, responseBody); match != nil {
		if s.markAccountBlacklisted(ctx, account, match) {
			return true
		}
	}

	runtimePlatform := EffectiveProtocol(account)
	customErrorCodesEnabled := account.IsCustomErrorCodesEnabled()

	if account.IsPoolMode() && !customErrorCodesEnabled {
		slog.Info("pool_mode_error_skipped", "account_id", account.ID, "status_code", statusCode)
		return false
	}
	if !account.ShouldHandleErrorCode(statusCode) {
		slog.Info("account_error_code_skipped", "account_id", account.ID, "status_code", statusCode)
		return false
	}
	if statusCode != 401 && s.tryTempUnschedulable(ctx, account, statusCode, responseBody) {
		return true
	}

	upstreamMsg := strings.TrimSpace(extractUpstreamErrorMessage(responseBody))
	upstreamMsg = sanitizeUpstreamErrorMessage(upstreamMsg)
	if upstreamMsg != "" {
		upstreamMsg = truncateForLog([]byte(upstreamMsg), 512)
	}

	switch statusCode {
	case 400:
		shouldDisable = s.handleBadRequestError(ctx, account, upstreamMsg)
	case 401:
		shouldDisable = s.handleUnauthorizedError(ctx, account, runtimePlatform, upstreamMsg, responseBody)
	case 402:
		msg := "Payment required (402): insufficient balance or billing issue"
		if upstreamMsg != "" {
			msg = "Payment required (402): " + upstreamMsg
		}
		s.handleAuthError(ctx, account, msg)
		shouldDisable = true
	case 403:
		logger.LegacyPrintf(
			"service.ratelimit",
			"[HandleUpstreamErrorRaw] account_id=%d platform=%s type=%s status=403 request_id=%s cf_ray=%s upstream_msg=%s raw_body=%s",
			account.ID,
			runtimePlatform,
			account.Type,
			strings.TrimSpace(headers.Get("x-request-id")),
			strings.TrimSpace(headers.Get("cf-ray")),
			upstreamMsg,
			truncateForLog(responseBody, 1024),
		)
		shouldDisable = s.handle403(ctx, account, upstreamMsg, responseBody)
	case 429:
		s.handle429(ctx, account, headers, responseBody)
	case 529:
		s.handle529(ctx, account)
	default:
		shouldDisable = s.handleDefaultUpstreamError(ctx, account, statusCode, upstreamMsg, customErrorCodesEnabled)
	}

	return shouldDisable
}

func (s *RateLimitService) handleBadRequestError(ctx context.Context, account *Account, upstreamMsg string) bool {
	msgLower := strings.ToLower(upstreamMsg)
	switch {
	case strings.Contains(msgLower, "organization has been disabled"):
		s.handleAuthError(ctx, account, "Organization disabled (400): "+upstreamMsg)
		return true
	case account.Platform == PlatformAnthropic && strings.Contains(msgLower, "credit balance"):
		s.handleAuthError(ctx, account, "Credit balance exhausted (400): "+upstreamMsg)
		return true
	case strings.Contains(msgLower, "identity verification is required"):
		s.handleAuthError(ctx, account, "Identity verification required (400): "+upstreamMsg)
		return true
	default:
		return false
	}
}

func (s *RateLimitService) handleUnauthorizedError(ctx context.Context, account *Account, runtimePlatform string, upstreamMsg string, responseBody []byte) bool {
	if runtimePlatform == PlatformOpenAI && isOpenAIPermanentUnauthorizedDetail(responseBody) {
		msg := "Unauthorized (401): account authentication failed permanently"
		if upstreamMsg != "" {
			msg = "Unauthorized (401): " + upstreamMsg
		}
		s.handleAuthError(ctx, account, msg)
		return true
	}

	if account.Type != AccountTypeOAuth || runtimePlatform == PlatformAntigravity {
		msg := "Authentication failed (401): invalid or expired credentials"
		if upstreamMsg != "" {
			msg = "Authentication failed (401): " + upstreamMsg
		}
		s.handleAuthError(ctx, account, msg)
		return true
	}

	if s.tokenCacheInvalidator != nil {
		if err := s.tokenCacheInvalidator.InvalidateToken(ctx, account); err != nil {
			slog.Warn("oauth_401_invalidate_cache_failed", "account_id", account.ID, "error", err)
		}
	}
	if account.Credentials == nil {
		account.Credentials = make(map[string]any)
	}
	account.Credentials["expires_at"] = time.Now().Format(time.RFC3339)
	if err := persistAccountCredentials(ctx, s.accountRepo, account, account.Credentials); err != nil {
		slog.Warn("oauth_401_force_refresh_update_failed", "account_id", account.ID, "error", err)
	} else {
		slog.Info("oauth_401_force_refresh_set", "account_id", account.ID, "platform", runtimePlatform)
	}

	msg := "Authentication failed (401): invalid or expired credentials"
	if upstreamMsg != "" {
		msg = "OAuth 401: " + upstreamMsg
	}
	cooldownMinutes := s.cfg.RateLimit.OAuth401CooldownMinutes
	if cooldownMinutes <= 0 {
		cooldownMinutes = 10
	}
	until := time.Now().Add(time.Duration(cooldownMinutes) * time.Minute)
	if err := s.accountRepo.SetTempUnschedulable(ctx, account.ID, until, msg); err != nil {
		slog.Warn("oauth_401_set_temp_unschedulable_failed", "account_id", account.ID, "error", err)
	}
	return true
}

func (s *RateLimitService) handleDefaultUpstreamError(ctx context.Context, account *Account, statusCode int, upstreamMsg string, customErrorCodesEnabled bool) bool {
	if customErrorCodesEnabled {
		msg := "Custom error code triggered"
		if upstreamMsg != "" {
			msg = upstreamMsg
		}
		s.handleCustomErrorCode(ctx, account, statusCode, msg)
		return true
	}
	if statusCode >= 500 {
		slog.Warn("account_upstream_error", "account_id", account.ID, "status_code", statusCode)
	}
	return false
}
