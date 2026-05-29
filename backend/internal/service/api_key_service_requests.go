package service

import "time"

type CreateAPIKeyRequest struct {
	Name             string                    `json:"name"`
	GroupID          *int64                    `json:"group_id"`
	Groups           *[]APIKeyGroupUpdateInput `json:"groups"`
	ModelDisplayMode *string                   `json:"model_display_mode"`
	CustomKey        *string                   `json:"custom_key"`   // 可选的自定义key
	IPWhitelist      []string                  `json:"ip_whitelist"` // IP 白名单
	IPBlacklist      []string                  `json:"ip_blacklist"` // IP 黑名单

	ImageOnlyEnabled         bool           `json:"image_only_enabled"`
	ImageCountBillingEnabled bool           `json:"image_count_billing_enabled"`
	ImageMaxCount            int            `json:"image_max_count"`
	ImageCountWeights        map[string]int `json:"image_count_weights"`

	Quota         float64 `json:"quota"`           // Quota limit in USD (0 = unlimited)
	ExpiresInDays *int    `json:"expires_in_days"` // Days until expiry (nil = never expires)

	RateLimit5h float64 `json:"rate_limit_5h"`
	RateLimit1d float64 `json:"rate_limit_1d"`
	RateLimit7d float64 `json:"rate_limit_7d"`
}

type UpdateAPIKeyRequest struct {
	Name             *string                   `json:"name"`
	GroupID          *int64                    `json:"group_id"`
	Groups           *[]APIKeyGroupUpdateInput `json:"groups"`
	ModelDisplayMode *string                   `json:"model_display_mode"`
	Status           *string                   `json:"status"`
	IPWhitelist      []string                  `json:"ip_whitelist"` // IP 白名单（空数组清空）
	IPBlacklist      []string                  `json:"ip_blacklist"` // IP 黑名单（空数组清空）

	ImageOnlyEnabled         *bool          `json:"image_only_enabled"`
	ImageCountBillingEnabled *bool          `json:"image_count_billing_enabled"`
	ImageMaxCount            *int           `json:"image_max_count"`
	ImageCountWeights        map[string]int `json:"image_count_weights"`

	Quota           *float64   `json:"quota"`       // Quota limit in USD (nil = no change, 0 = unlimited)
	ExpiresAt       *time.Time `json:"expires_at"`  // Expiration time (nil = no change)
	ClearExpiration bool       `json:"-"`           // Clear expiration (internal use)
	ResetQuota      *bool      `json:"reset_quota"` // Reset quota_used to 0

	RateLimit5h         *float64 `json:"rate_limit_5h"`
	RateLimit1d         *float64 `json:"rate_limit_1d"`
	RateLimit7d         *float64 `json:"rate_limit_7d"`
	ResetRateLimitUsage *bool    `json:"reset_rate_limit_usage"` // Reset all usage counters to 0
}
