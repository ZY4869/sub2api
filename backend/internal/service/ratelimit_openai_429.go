package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strconv"
	"time"
)

// calculateOpenAI429ResetTime 从 OpenAI 429 响应头计算正确的重置时间
// 返回 nil 表示无法从响应头中确定重置时间
func (s *RateLimitService) calculateOpenAI429ResetTime(headers http.Header) *time.Time {
	snapshot := ParseCodexRateLimitHeaders(headers)
	if snapshot == nil {
		return nil
	}

	normalized := snapshot.Normalize()
	if normalized == nil {
		return nil
	}

	now := time.Now()
	is7dExhausted := normalized.Used7dPercent != nil && *normalized.Used7dPercent >= 100
	is5hExhausted := normalized.Used5hPercent != nil && *normalized.Used5hPercent >= 100

	if is7dExhausted && normalized.Reset7dSeconds != nil {
		resetAt := now.Add(time.Duration(*normalized.Reset7dSeconds) * time.Second)
		slog.Info("openai_429_7d_limit_exhausted", "reset_after_seconds", *normalized.Reset7dSeconds, "reset_at", resetAt)
		return &resetAt
	}
	if is5hExhausted && normalized.Reset5hSeconds != nil {
		resetAt := now.Add(time.Duration(*normalized.Reset5hSeconds) * time.Second)
		slog.Info("openai_429_5h_limit_exhausted", "reset_after_seconds", *normalized.Reset5hSeconds, "reset_at", resetAt)
		return &resetAt
	}

	var maxResetSecs int
	if normalized.Reset7dSeconds != nil && *normalized.Reset7dSeconds > maxResetSecs {
		maxResetSecs = *normalized.Reset7dSeconds
	}
	if normalized.Reset5hSeconds != nil && *normalized.Reset5hSeconds > maxResetSecs {
		maxResetSecs = *normalized.Reset5hSeconds
	}
	if maxResetSecs > 0 {
		resetAt := now.Add(time.Duration(maxResetSecs) * time.Second)
		slog.Info("openai_429_using_max_reset", "max_reset_seconds", maxResetSecs, "reset_at", resetAt)
		return &resetAt
	}

	return nil
}

func (s *RateLimitService) persistOpenAICodexSnapshot(ctx context.Context, account *Account, headers http.Header) *openAICodexRateLimitState {
	if s == nil || s.accountRepo == nil || account == nil || headers == nil {
		return nil
	}
	snapshot := ParseCodexRateLimitHeaders(headers)
	if snapshot == nil {
		return nil
	}
	scope, ok := resolveOpenAICodexSnapshotScopeFromContext(ctx, account)
	if !ok {
		return nil
	}
	ctx = withOpenAICodexResolvedQuotaScope(ctx, scope)
	slog.Info(
		"openai_codex_snapshot_scope_resolved",
		"account_id", account.ID,
		"requested_model", openAICodexRequestModelFromContext(ctx),
		"upstream_model", openAICodexRequestModelFromContext(ctx),
		"resolved_scope", scope,
		"snapshot_source", "429",
	)
	updates := buildCodexUsageExtraUpdatesForScope(scope, snapshot, time.Now())
	if len(updates) == 0 {
		return nil
	}
	return syncOpenAICodexRateLimitState(ctx, s.accountRepo, account, updates, time.Now())
}

// parseOpenAIRateLimitResetTime 解析 OpenAI 格式的 429 响应，返回重置时间的 Unix 时间戳
func parseOpenAIRateLimitResetTime(body []byte) *int64 {
	var parsed map[string]any
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil
	}

	errObj, ok := parsed["error"].(map[string]any)
	if !ok {
		return nil
	}
	errType, _ := errObj["type"].(string)
	if errType != "usage_limit_reached" && errType != "rate_limit_exceeded" {
		return nil
	}

	if resetsAt, ok := errObj["resets_at"].(float64); ok {
		ts := int64(resetsAt)
		return &ts
	}
	if resetsAt, ok := errObj["resets_at"].(string); ok {
		if ts, err := strconv.ParseInt(resetsAt, 10, 64); err == nil {
			return &ts
		}
	}
	if resetsInSeconds, ok := errObj["resets_in_seconds"].(float64); ok {
		ts := time.Now().Unix() + int64(resetsInSeconds)
		return &ts
	}
	if resetsInSeconds, ok := errObj["resets_in_seconds"].(string); ok {
		if sec, err := strconv.ParseInt(resetsInSeconds, 10, 64); err == nil {
			ts := time.Now().Unix() + sec
			return &ts
		}
	}

	return nil
}
