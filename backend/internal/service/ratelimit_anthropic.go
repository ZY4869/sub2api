package service

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const (
	anthropicFableRateLimitKey = "anthropic:7d_oi:fable"
	anthropicFableWindowReason = "anthropic_7d_oi_window_exhausted"
)

type anthropic429Result struct {
	resetAt       time.Time  // The correct reset time to use for SetRateLimited
	fiveHourReset *time.Time // 5h window reset timestamp (for session window calculation), nil if not available
	reason        string
}

type anthropicWindowLimit struct {
	window  string
	resetAt time.Time
	reason  string
}

func (s *RateLimitService) persistAnthropicFableWindowLimit(ctx context.Context, account *Account, headers http.Header) bool {
	if s == nil || s.accountRepo == nil || account == nil {
		return false
	}
	limit := selectAnthropicFableWindowLimit(headers, time.Now())
	if limit == nil {
		return false
	}
	s.samplePassiveUsageFromHeaders(ctx, account, headers)
	if err := s.accountRepo.SetModelRateLimit(ctx, account.ID, anthropicFableRateLimitKey, limit.resetAt); err != nil {
		slog.Warn("anthropic_fable_window_rate_limit_set_failed", "account_id", account.ID, "scope", anthropicFableRateLimitKey, "reset_at", limit.resetAt, "error", err)
		return true
	}
	slog.Info("anthropic_fable_window_model_rate_limited", "account_id", account.ID, "scope", anthropicFableRateLimitKey, "reason", limit.reason, "reset_at", limit.resetAt, "reset_in", time.Until(limit.resetAt).Truncate(time.Second))
	return true
}

func selectAnthropicFableWindowLimit(headers http.Header, now time.Time) *anthropicWindowLimit {
	if !isAnthropicWindowRejected(headers, "7d_oi") && !isAnthropicWindowExceeded(headers, "7d_oi") {
		return nil
	}
	resetAt, ok := parseAnthropicWindowReset(headers, "7d_oi", now)
	if !ok {
		resetAt, ok = parseAnthropicAggregateReset(headers, now)
	}
	if !ok {
		return nil
	}
	return &anthropicWindowLimit{
		window:  "7d_oi",
		resetAt: resetAt,
		reason:  anthropicFableWindowReason,
	}
}

// calculateAnthropic429ResetTime parses Anthropic's per-window rate-limit headers
// to determine which window (5h or 7d) actually triggered the 429.
//
// Headers used:
//   - anthropic-ratelimit-unified-5h-utilization / anthropic-ratelimit-unified-5h-surpassed-threshold
//   - anthropic-ratelimit-unified-5h-reset
//   - anthropic-ratelimit-unified-7d-utilization / anthropic-ratelimit-unified-7d-surpassed-threshold
//   - anthropic-ratelimit-unified-7d-reset
//
// Returns nil when the per-window headers are absent (caller should fall back to
// the aggregated anthropic-ratelimit-unified-reset header).
func calculateAnthropic429ResetTime(headers http.Header) *anthropic429Result {
	reset5hStr := headers.Get("anthropic-ratelimit-unified-5h-reset")
	reset7dStr := headers.Get("anthropic-ratelimit-unified-7d-reset")

	if reset5hStr == "" && reset7dStr == "" {
		return nil
	}

	var reset5h, reset7d *time.Time
	if ts, err := strconv.ParseInt(reset5hStr, 10, 64); err == nil {
		t := time.Unix(ts, 0)
		reset5h = &t
	}
	if ts, err := strconv.ParseInt(reset7dStr, 10, 64); err == nil {
		t := time.Unix(ts, 0)
		reset7d = &t
	}

	is5hExceeded := isAnthropicWindowExceeded(headers, "5h")
	is7dExceeded := isAnthropicWindowExceeded(headers, "7d")

	slog.Info("anthropic_429_window_analysis",
		"is_5h_exceeded", is5hExceeded,
		"is_7d_exceeded", is7dExceeded,
		"reset_5h", reset5hStr,
		"reset_7d", reset7dStr,
	)

	// Select the correct reset time based on which window(s) are exceeded.
	var chosen *time.Time
	switch {
	case is5hExceeded && is7dExceeded:
		// Both exceeded → prefer 7d (longer cooldown), fall back to 5h
		chosen = reset7d
		if chosen == nil {
			chosen = reset5h
		}
	case is5hExceeded:
		chosen = reset5h
	case is7dExceeded:
		chosen = reset7d
	default:
		// Neither flag clearly exceeded — pick the sooner reset as best guess
		chosen = pickSooner(reset5h, reset7d)
	}

	if chosen == nil {
		return nil
	}
	reason := AccountRateLimitReason429
	switch {
	case is7dExceeded:
		reason = AccountRateLimitReasonUsage7d
	case is5hExceeded:
		reason = AccountRateLimitReasonUsage5h
	}
	return &anthropic429Result{resetAt: *chosen, fiveHourReset: reset5h, reason: reason}
}

// isAnthropicWindowExceeded checks whether a given Anthropic rate-limit window
// (e.g. "5h" or "7d") has been exceeded, using utilization and surpassed-threshold headers.
func isAnthropicWindowExceeded(headers http.Header, window string) bool {
	prefix := "anthropic-ratelimit-unified-" + window + "-"

	// Check surpassed-threshold first (most explicit signal)
	if st := headers.Get(prefix + "surpassed-threshold"); strings.EqualFold(st, "true") {
		return true
	}

	// Fall back to utilization >= 1.0
	if utilStr := headers.Get(prefix + "utilization"); utilStr != "" {
		if util, err := strconv.ParseFloat(utilStr, 64); err == nil && util >= 1.0-1e-9 {
			// Use a small epsilon to handle floating point: treat 0.9999999... as >= 1.0
			return true
		}
	}

	return false
}

func isAnthropicWindowRejected(headers http.Header, window string) bool {
	return strings.EqualFold(headers.Get("anthropic-ratelimit-unified-"+window+"-status"), "rejected")
}

func parseAnthropicWindowReset(headers http.Header, window string, now time.Time) (time.Time, bool) {
	return parseAnthropicResetTimestamp(headers.Get("anthropic-ratelimit-unified-"+window+"-reset"), now, 8*24*time.Hour)
}

func parseAnthropicAggregateReset(headers http.Header, now time.Time) (time.Time, bool) {
	return parseAnthropicResetTimestamp(headers.Get("anthropic-ratelimit-unified-reset"), now, 8*24*time.Hour)
}

func parseAnthropicResetTimestamp(raw string, now time.Time, maxFuture time.Duration) (time.Time, bool) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return time.Time{}, false
	}
	ts, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, false
	}
	if ts > 1e11 {
		ts /= 1000
	}
	resetAt := time.Unix(ts, 0)
	if resetAt.Before(now.Add(-time.Hour)) || resetAt.After(now.Add(maxFuture)) {
		return time.Time{}, false
	}
	return resetAt, true
}

// pickSooner returns whichever of the two time pointers is earlier.
// If only one is non-nil, it is returned. If both are nil, returns nil.
func pickSooner(a, b *time.Time) *time.Time {
	switch {
	case a != nil && b != nil:
		if a.Before(*b) {
			return a
		}
		return b
	case a != nil:
		return a
	default:
		return b
	}
}
