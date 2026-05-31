package service

import (
	"context"
	"time"
)

func (s *APIKeyService) applyAPIKeyUpdateFields(ctx context.Context, apiKey *APIKey, req UpdateAPIKeyRequest) bool {
	if req.Name != nil {
		apiKey.Name = *req.Name
	}
	if req.ModelDisplayMode != nil {
		apiKey.ModelDisplayMode = NormalizeAPIKeyModelDisplayMode(*req.ModelDisplayMode)
	}
	if req.Status != nil {
		apiKey.Status = *req.Status
		if s.cache != nil {
			_ = s.cache.DeleteCreateAttemptCount(ctx, apiKey.UserID)
		}
	}

	applyAPIKeyUpdateQuotaFields(apiKey, req)
	applyAPIKeyUpdateTimeAccessFields(apiKey, req)
	applyAPIKeyUpdateImageFields(apiKey, req)
	applyAPIKeyUpdateRateLimitFields(apiKey, req)

	apiKey.IPWhitelist = req.IPWhitelist
	apiKey.IPBlacklist = req.IPBlacklist
	resetRateLimit := req.ResetRateLimitUsage != nil && *req.ResetRateLimitUsage
	if resetRateLimit {
		apiKey.Usage5h = 0
		apiKey.Usage1d = 0
		apiKey.Usage7d = 0
		apiKey.Window5hStart = nil
		apiKey.Window1dStart = nil
		apiKey.Window7dStart = nil
	}
	return resetRateLimit
}

func applyAPIKeyUpdateTimeAccessFields(apiKey *APIKey, req UpdateAPIKeyRequest) {
	if req.ClearStartsAt {
		apiKey.StartsAt = nil
	} else if req.StartsAt != nil {
		apiKey.StartsAt = req.StartsAt
	}
	if req.ClearAccessTimePolicy {
		apiKey.AccessTimePolicy = nil
	} else if req.AccessTimePolicy != nil {
		apiKey.AccessTimePolicy = req.AccessTimePolicy
	}
}

func applyAPIKeyUpdateQuotaFields(apiKey *APIKey, req UpdateAPIKeyRequest) {
	if req.Quota != nil {
		apiKey.Quota = *req.Quota
		if apiKey.Status == StatusAPIKeyQuotaExhausted && *req.Quota > apiKey.QuotaUsed {
			apiKey.Status = StatusActive
		}
	}
	if req.ResetQuota != nil && *req.ResetQuota {
		apiKey.QuotaUsed = 0
		if apiKey.Status == StatusAPIKeyQuotaExhausted {
			apiKey.Status = StatusActive
		}
	}
	if req.ClearExpiration {
		apiKey.ExpiresAt = nil
		if apiKey.Status == StatusAPIKeyExpired {
			apiKey.Status = StatusActive
		}
	} else if req.ExpiresAt != nil {
		apiKey.ExpiresAt = req.ExpiresAt
		if apiKey.Status == StatusAPIKeyExpired && time.Now().Before(*req.ExpiresAt) {
			apiKey.Status = StatusActive
		}
	}
}

func applyAPIKeyUpdateImageFields(apiKey *APIKey, req UpdateAPIKeyRequest) {
	if req.ImageOnlyEnabled != nil {
		apiKey.ImageOnlyEnabled = *req.ImageOnlyEnabled
	}
	if req.ImageCountBillingEnabled != nil {
		apiKey.ImageCountBillingEnabled = *req.ImageCountBillingEnabled
	}
	if req.ImageMaxCount != nil {
		apiKey.ImageMaxCount = max(*req.ImageMaxCount, 0)
	}
	if req.ImageCountWeights != nil {
		apiKey.ImageCountWeights = NormalizeAPIKeyImageCountWeights(req.ImageCountWeights)
	}
	if !apiKey.ImageOnlyEnabled || !apiKey.ImageCountBillingEnabled || apiKey.ImageMaxCount <= 0 {
		apiKey.ImageCountBillingEnabled = false
		apiKey.ImageMaxCount = 0
	}
}

func applyAPIKeyUpdateRateLimitFields(apiKey *APIKey, req UpdateAPIKeyRequest) {
	if req.RateLimit5h != nil {
		apiKey.RateLimit5h = *req.RateLimit5h
	}
	if req.RateLimit1d != nil {
		apiKey.RateLimit1d = *req.RateLimit1d
	}
	if req.RateLimit7d != nil {
		apiKey.RateLimit7d = *req.RateLimit7d
	}
}
