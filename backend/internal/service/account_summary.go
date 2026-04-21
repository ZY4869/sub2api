package service

type AccountStatusSummaryFilters struct {
	Platform      string
	AccountType   string
	Search        string
	GroupID       int64
	Lifecycle     string
	PrivacyMode   string
	LimitedView   string
	LimitedReason string
	RuntimeView   string
}

type AccountLimitedBreakdown struct {
	Total      int64 `json:"total"`
	Rate429    int64 `json:"rate_429"`
	Usage5h    int64 `json:"usage_5h"`
	Usage7d    int64 `json:"usage_7d"`
	Usage7dAll int64 `json:"usage_7d_all"`
}

type AccountStatusSummary struct {
	Total              int64                   `json:"total"`
	ByStatus           map[string]int64        `json:"by_status"`
	RateLimited        int64                   `json:"rate_limited"`
	TempUnschedulable  int64                   `json:"temp_unschedulable"`
	Overloaded         int64                   `json:"overloaded"`
	Paused             int64                   `json:"paused"`
	InUse              int64                   `json:"in_use"`
	RemainingAvailable int64                   `json:"remaining_available"`
	ByPlatform         map[string]int64        `json:"by_platform"`
	LimitedBreakdown   AccountLimitedBreakdown `json:"limited_breakdown"`
	DispatchableCount  int64                   `json:"-"`
}
