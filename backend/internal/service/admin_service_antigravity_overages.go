package service

import "strings"

func sanitizeAntigravityOveragesExtra(platform string, extra map[string]any) {
	if !strings.EqualFold(strings.TrimSpace(platform), PlatformAntigravity) || len(extra) == 0 {
		return
	}
	if enabled, _ := extra["allow_overages"].(bool); enabled {
		delete(extra, modelRateLimitsKey)
		return
	}
	rawLimits, ok := extra[modelRateLimitsKey].(map[string]any)
	if !ok || len(rawLimits) == 0 {
		return
	}
	if _, exists := rawLimits[creditsExhaustedKey]; !exists {
		return
	}
	delete(rawLimits, creditsExhaustedKey)
	if len(rawLimits) == 0 {
		delete(extra, modelRateLimitsKey)
		return
	}
	extra[modelRateLimitsKey] = rawLimits
}
