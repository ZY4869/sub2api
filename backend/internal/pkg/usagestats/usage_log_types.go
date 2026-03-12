// Package usagestats provides types for usage statistics and reporting.
package usagestats

import "time"

// DashboardStats represents admin dashboard statistics.
type DashboardStats struct {
	TotalUsers               int64   `json:"total_users"`
	TodayNewUsers            int64   `json:"today_new_users"`
	ActiveUsers              int64   `json:"active_users"`
	HourlyActiveUsers        int64   `json:"hourly_active_users"`
	StatsUpdatedAt           string  `json:"stats_updated_at"`
	StatsStale               bool    `json:"stats_stale"`
	TotalAPIKeys             int64   `json:"total_api_keys"`
	ActiveAPIKeys            int64   `json:"active_api_keys"`
	TotalAccounts            int64   `json:"total_accounts"`
	NormalAccounts           int64   `json:"normal_accounts"`
	ErrorAccounts            int64   `json:"error_accounts"`
	RateLimitAccounts        int64   `json:"ratelimit_accounts"`
	OverloadAccounts         int64   `json:"overload_accounts"`
	TotalRequests            int64   `json:"total_requests"`
	TotalInputTokens         int64   `json:"total_input_tokens"`
	TotalOutputTokens        int64   `json:"total_output_tokens"`
	TotalCacheCreationTokens int64   `json:"total_cache_creation_tokens"`
	TotalCacheReadTokens     int64   `json:"total_cache_read_tokens"`
	TotalTokens              int64   `json:"total_tokens"`
	TotalCost                float64 `json:"total_cost"`
	TotalActualCost          float64 `json:"total_actual_cost"`
	AdminFreeRequests        int64   `json:"admin_free_requests"`
	AdminFreeStandardCost    float64 `json:"admin_free_standard_cost"`
	TodayRequests            int64   `json:"today_requests"`
	TodayInputTokens         int64   `json:"today_input_tokens"`
	TodayOutputTokens        int64   `json:"today_output_tokens"`
	TodayCacheCreationTokens int64   `json:"today_cache_creation_tokens"`
	TodayCacheReadTokens     int64   `json:"today_cache_read_tokens"`
	TodayTokens              int64   `json:"today_tokens"`
	TodayCost                float64 `json:"today_cost"`
	TodayActualCost          float64 `json:"today_actual_cost"`
	AverageDurationMs        float64 `json:"average_duration_ms"`
	Rpm                      int64   `json:"rpm"`
	Tpm                      int64   `json:"tpm"`
}

// TrendDataPoint represents a single point in trend data.
type TrendDataPoint struct {
	Date                string  `json:"date"`
	Requests            int64   `json:"requests"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Cost                float64 `json:"cost"`
	ActualCost          float64 `json:"actual_cost"`
}

// ModelStat represents usage statistics for a single model.
type ModelStat struct {
	Model               string  `json:"model"`
	Requests            int64   `json:"requests"`
	InputTokens         int64   `json:"input_tokens"`
	OutputTokens        int64   `json:"output_tokens"`
	CacheCreationTokens int64   `json:"cache_creation_tokens"`
	CacheReadTokens     int64   `json:"cache_read_tokens"`
	TotalTokens         int64   `json:"total_tokens"`
	Cost                float64 `json:"cost"`
	ActualCost          float64 `json:"actual_cost"`
}

// GroupStat represents usage statistics for a single group.
type GroupStat struct {
	GroupID     int64   `json:"group_id"`
	GroupName   string  `json:"group_name"`
	Requests    int64   `json:"requests"`
	TotalTokens int64   `json:"total_tokens"`
	Cost        float64 `json:"cost"`
	ActualCost  float64 `json:"actual_cost"`
}

// UserUsageTrendPoint represents user usage trend data point.
type UserUsageTrendPoint struct {
	Date       string  `json:"date"`
	UserID     int64   `json:"user_id"`
	Email      string  `json:"email"`
	Requests   int64   `json:"requests"`
	Tokens     int64   `json:"tokens"`
	Cost       float64 `json:"cost"`
	ActualCost float64 `json:"actual_cost"`
}

// APIKeyUsageTrendPoint represents API key usage trend data point.
type APIKeyUsageTrendPoint struct {
	Date     string `json:"date"`
	APIKeyID int64  `json:"api_key_id"`
	KeyName  string `json:"key_name"`
	Requests int64  `json:"requests"`
	Tokens   int64  `json:"tokens"`
}

// UserDashboardStats represents per-user dashboard statistics.
type UserDashboardStats struct {
	TotalAPIKeys             int64   `json:"total_api_keys"`
	ActiveAPIKeys            int64   `json:"active_api_keys"`
	TotalRequests            int64   `json:"total_requests"`
	TotalInputTokens         int64   `json:"total_input_tokens"`
	TotalOutputTokens        int64   `json:"total_output_tokens"`
	TotalCacheCreationTokens int64   `json:"total_cache_creation_tokens"`
	TotalCacheReadTokens     int64   `json:"total_cache_read_tokens"`
	TotalTokens              int64   `json:"total_tokens"`
	TotalCost                float64 `json:"total_cost"`
	TotalActualCost          float64 `json:"total_actual_cost"`
	AdminFreeRequests        int64   `json:"admin_free_requests"`
	AdminFreeStandardCost    float64 `json:"admin_free_standard_cost"`
	TodayRequests            int64   `json:"today_requests"`
	TodayInputTokens         int64   `json:"today_input_tokens"`
	TodayOutputTokens        int64   `json:"today_output_tokens"`
	TodayCacheCreationTokens int64   `json:"today_cache_creation_tokens"`
	TodayCacheReadTokens     int64   `json:"today_cache_read_tokens"`
	TodayTokens              int64   `json:"today_tokens"`
	TodayCost                float64 `json:"today_cost"`
	TodayActualCost          float64 `json:"today_actual_cost"`
	AverageDurationMs        float64 `json:"average_duration_ms"`
	Rpm                      int64   `json:"rpm"`
	Tpm                      int64   `json:"tpm"`
}

// UsageLogFilters represents filters for usage log queries.
type UsageLogFilters struct {
	UserID      int64
	APIKeyID    int64
	AccountID   int64
	GroupID     int64
	Model       string
	RequestType *int16
	Stream      *bool
	BillingType *int8
	StartTime   *time.Time
	EndTime     *time.Time
	ExactTotal  bool
}

// UsageStats represents usage statistics.
type UsageStats struct {
	TotalRequests         int64    `json:"total_requests"`
	TotalInputTokens      int64    `json:"total_input_tokens"`
	TotalOutputTokens     int64    `json:"total_output_tokens"`
	TotalCacheTokens      int64    `json:"total_cache_tokens"`
	TotalTokens           int64    `json:"total_tokens"`
	TotalCost             float64  `json:"total_cost"`
	TotalActualCost       float64  `json:"total_actual_cost"`
	AdminFreeRequests     int64    `json:"admin_free_requests"`
	AdminFreeStandardCost float64  `json:"admin_free_standard_cost"`
	TotalAccountCost      *float64 `json:"total_account_cost,omitempty"`
	AverageDurationMs     float64  `json:"average_duration_ms"`
}

// BatchUserUsageStats represents usage stats for a single user.
type BatchUserUsageStats struct {
	UserID          int64   `json:"user_id"`
	TodayActualCost float64 `json:"today_actual_cost"`
	TotalActualCost float64 `json:"total_actual_cost"`
}

// BatchAPIKeyUsageStats represents usage stats for a single API key.
type BatchAPIKeyUsageStats struct {
	APIKeyID        int64   `json:"api_key_id"`
	TodayActualCost float64 `json:"today_actual_cost"`
	TotalActualCost float64 `json:"total_actual_cost"`
}

// AccountUsageHistory represents daily usage history for an account.
type AccountUsageHistory struct {
	Date       string  `json:"date"`
	Label      string  `json:"label"`
	Requests   int64   `json:"requests"`
	Tokens     int64   `json:"tokens"`
	Cost       float64 `json:"cost"`
	ActualCost float64 `json:"actual_cost"`
	UserCost   float64 `json:"user_cost"`
}

// AccountUsageSummary represents summary statistics for an account.
type AccountUsageSummary struct {
	Days              int     `json:"days"`
	ActualDaysUsed    int     `json:"actual_days_used"`
	TotalCost         float64 `json:"total_cost"`
	TotalUserCost     float64 `json:"total_user_cost"`
	TotalStandardCost float64 `json:"total_standard_cost"`
	TotalRequests     int64   `json:"total_requests"`
	TotalTokens       int64   `json:"total_tokens"`
	AvgDailyCost      float64 `json:"avg_daily_cost"`
	AvgDailyUserCost  float64 `json:"avg_daily_user_cost"`
	AvgDailyRequests  float64 `json:"avg_daily_requests"`
	AvgDailyTokens    float64 `json:"avg_daily_tokens"`
	AvgDurationMs     float64 `json:"avg_duration_ms"`
	Today             *struct {
		Date     string  `json:"date"`
		Cost     float64 `json:"cost"`
		UserCost float64 `json:"user_cost"`
		Requests int64   `json:"requests"`
		Tokens   int64   `json:"tokens"`
	} `json:"today"`
	HighestCostDay *struct {
		Date     string  `json:"date"`
		Label    string  `json:"label"`
		Cost     float64 `json:"cost"`
		UserCost float64 `json:"user_cost"`
		Requests int64   `json:"requests"`
	} `json:"highest_cost_day"`
	HighestRequestDay *struct {
		Date     string  `json:"date"`
		Label    string  `json:"label"`
		Requests int64   `json:"requests"`
		Cost     float64 `json:"cost"`
		UserCost float64 `json:"user_cost"`
	} `json:"highest_request_day"`
}

// AccountUsageStatsResponse represents the full usage statistics response for an account.
type AccountUsageStatsResponse struct {
	History []AccountUsageHistory `json:"history"`
	Summary AccountUsageSummary   `json:"summary"`
	Models  []ModelStat           `json:"models"`
}
