package service

import (
	"strconv"
	"strings"
	"time"

	"github.com/tidwall/gjson"
)

func parseCodexRateLimitsFromWSMessage(message []byte, now time.Time) *OpenAICodexUsageSnapshot {
	if len(message) == 0 {
		return nil
	}
	basePaths := []string{"rate_limits", "payload.rate_limits", "payload.rate_limits.rate_limits"}
	for _, base := range basePaths {
		snapshot := &OpenAICodexUsageSnapshot{}
		hasData := false

		if v, ok := parseCodexWSRateLimitFloat(message, base+".primary.used_percent"); ok {
			snapshot.PrimaryUsedPercent = &v
			hasData = true
		}
		if v, ok := parseCodexWSRateLimitInt(message, base+".primary.window_minutes"); ok {
			snapshot.PrimaryWindowMinutes = &v
			hasData = true
		}
		if v, ok := parseCodexWSRateLimitResetAfterSeconds(message, base+".primary", now); ok {
			snapshot.PrimaryResetAfterSeconds = &v
			hasData = true
		}

		if v, ok := parseCodexWSRateLimitFloat(message, base+".secondary.used_percent"); ok {
			snapshot.SecondaryUsedPercent = &v
			hasData = true
		}
		if v, ok := parseCodexWSRateLimitInt(message, base+".secondary.window_minutes"); ok {
			snapshot.SecondaryWindowMinutes = &v
			hasData = true
		}
		if v, ok := parseCodexWSRateLimitResetAfterSeconds(message, base+".secondary", now); ok {
			snapshot.SecondaryResetAfterSeconds = &v
			hasData = true
		}

		if !hasData {
			continue
		}
		snapshot.UpdatedAt = now.UTC().Format(time.RFC3339)
		return snapshot
	}
	return nil
}

func parseCodexWSRateLimitFloat(message []byte, path string) (float64, bool) {
	value := gjson.GetBytes(message, path)
	if !value.Exists() {
		return 0, false
	}
	if value.Type == gjson.Number {
		return value.Float(), true
	}
	if value.Type == gjson.String {
		raw := strings.TrimSpace(value.String())
		if raw == "" {
			return 0, false
		}
		if f, err := strconv.ParseFloat(raw, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

func parseCodexWSRateLimitInt(message []byte, path string) (int, bool) {
	value := gjson.GetBytes(message, path)
	if !value.Exists() {
		return 0, false
	}
	if value.Type == gjson.Number {
		return int(value.Int()), true
	}
	if value.Type == gjson.String {
		raw := strings.TrimSpace(value.String())
		if raw == "" {
			return 0, false
		}
		if i, err := strconv.Atoi(raw); err == nil {
			return i, true
		}
	}
	return 0, false
}

func parseCodexWSRateLimitResetAfterSeconds(message []byte, base string, now time.Time) (int, bool) {
	if v, ok := parseCodexWSRateLimitInt(message, base+".resets_in_seconds"); ok {
		if v < 0 {
			v = 0
		}
		return v, true
	}
	resetAt := gjson.GetBytes(message, base+".resets_at")
	if !resetAt.Exists() {
		return 0, false
	}
	epochSeconds, ok := parseCodexWSUnixSeconds(resetAt)
	if !ok {
		return 0, false
	}
	resetTime := time.Unix(epochSeconds, 0).UTC()
	sec := int(resetTime.Sub(now).Seconds())
	if sec < 0 {
		sec = 0
	}
	return sec, true
}

func parseCodexWSUnixSeconds(value gjson.Result) (int64, bool) {
	switch value.Type {
	case gjson.Number:
		epoch := value.Int()
		if epoch <= 0 {
			return 0, false
		}
		// Some upstreams may send milliseconds.
		if epoch > 2_000_000_000_000 {
			epoch /= 1000
		}
		return epoch, true
	case gjson.String:
		raw := strings.TrimSpace(value.String())
		if raw == "" {
			return 0, false
		}
		epoch, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || epoch <= 0 {
			return 0, false
		}
		if epoch > 2_000_000_000_000 {
			epoch /= 1000
		}
		return epoch, true
	default:
		return 0, false
	}
}
