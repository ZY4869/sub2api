package config

import "time"

type OpsConfig struct {
	Enabled                bool                           `mapstructure:"enabled"`
	UsePreaggregatedTables bool                           `mapstructure:"use_preaggregated_tables"`
	Cleanup                OpsCleanupConfig               `mapstructure:"cleanup"`
	MetricsCollectorCache  OpsMetricsCollectorCacheConfig `mapstructure:"metrics_collector_cache"`
	Aggregation            OpsAggregationConfig           `mapstructure:"aggregation"`
}
type OpsCleanupConfig struct {
	Enabled                    bool   `mapstructure:"enabled"`
	Schedule                   string `mapstructure:"schedule"`
	ErrorLogRetentionDays      int    `mapstructure:"error_log_retention_days"`
	MinuteMetricsRetentionDays int    `mapstructure:"minute_metrics_retention_days"`
	HourlyMetricsRetentionDays int    `mapstructure:"hourly_metrics_retention_days"`
}
type OpsAggregationConfig struct {
	Enabled bool `mapstructure:"enabled"`
}
type OpsMetricsCollectorCacheConfig struct {
	Enabled bool          `mapstructure:"enabled"`
	TTL     time.Duration `mapstructure:"ttl"`
}
type JWTConfig struct {
	Secret                   string `mapstructure:"secret"`
	ExpireHour               int    `mapstructure:"expire_hour"`
	AccessTokenExpireMinutes int    `mapstructure:"access_token_expire_minutes"`
	RefreshTokenExpireDays   int    `mapstructure:"refresh_token_expire_days"`
	RefreshWindowMinutes     int    `mapstructure:"refresh_window_minutes"`
}
type TotpConfig struct {
	EncryptionKey           string `mapstructure:"encryption_key"`
	EncryptionKeyConfigured bool   `mapstructure:"-"`
}
type TurnstileConfig struct {
	Required bool `mapstructure:"required"`
}
type DefaultConfig struct {
	AdminEmail      string  `mapstructure:"admin_email"`
	AdminPassword   string  `mapstructure:"admin_password"`
	UserConcurrency int     `mapstructure:"user_concurrency"`
	UserBalance     float64 `mapstructure:"user_balance"`
	APIKeyPrefix    string  `mapstructure:"api_key_prefix"`
	RateMultiplier  float64 `mapstructure:"rate_multiplier"`
}
type RateLimitConfig struct {
	OverloadCooldownMinutes int `mapstructure:"overload_cooldown_minutes"`
	OAuth401CooldownMinutes int `mapstructure:"oauth_401_cooldown_minutes"`
}
type APIKeyAuthCacheConfig struct {
	L1Size             int  `mapstructure:"l1_size"`
	L1TTLSeconds       int  `mapstructure:"l1_ttl_seconds"`
	L2TTLSeconds       int  `mapstructure:"l2_ttl_seconds"`
	NegativeTTLSeconds int  `mapstructure:"negative_ttl_seconds"`
	JitterPercent      int  `mapstructure:"jitter_percent"`
	Singleflight       bool `mapstructure:"singleflight"`
}
type SubscriptionCacheConfig struct {
	L1Size        int `mapstructure:"l1_size"`
	L1TTLSeconds  int `mapstructure:"l1_ttl_seconds"`
	JitterPercent int `mapstructure:"jitter_percent"`
}
type SubscriptionMaintenanceConfig struct {
	WorkerCount int `mapstructure:"worker_count"`
	QueueSize   int `mapstructure:"queue_size"`
}
type DashboardCacheConfig struct {
	Enabled                    bool   `mapstructure:"enabled"`
	KeyPrefix                  string `mapstructure:"key_prefix"`
	StatsFreshTTLSeconds       int    `mapstructure:"stats_fresh_ttl_seconds"`
	StatsTTLSeconds            int    `mapstructure:"stats_ttl_seconds"`
	StatsRefreshTimeoutSeconds int    `mapstructure:"stats_refresh_timeout_seconds"`
}
type DashboardAggregationConfig struct {
	Enabled         bool                                `mapstructure:"enabled"`
	IntervalSeconds int                                 `mapstructure:"interval_seconds"`
	LookbackSeconds int                                 `mapstructure:"lookback_seconds"`
	BackfillEnabled bool                                `mapstructure:"backfill_enabled"`
	BackfillMaxDays int                                 `mapstructure:"backfill_max_days"`
	Retention       DashboardAggregationRetentionConfig `mapstructure:"retention"`
	RecomputeDays   int                                 `mapstructure:"recompute_days"`
}
type DashboardAggregationRetentionConfig struct {
	UsageLogsDays         int `mapstructure:"usage_logs_days"`
	UsageBillingDedupDays int `mapstructure:"usage_billing_dedup_days"`
	HourlyDays            int `mapstructure:"hourly_days"`
	DailyDays             int `mapstructure:"daily_days"`
}
type UsageCleanupConfig struct {
	Enabled               bool `mapstructure:"enabled"`
	MaxRangeDays          int  `mapstructure:"max_range_days"`
	BatchSize             int  `mapstructure:"batch_size"`
	WorkerIntervalSeconds int  `mapstructure:"worker_interval_seconds"`
	TaskTimeoutSeconds    int  `mapstructure:"task_timeout_seconds"`
}
