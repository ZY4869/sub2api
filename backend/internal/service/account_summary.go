package service

type AccountStatusSummaryFilters struct {
	Platform    string
	AccountType string
	Search      string
	GroupID     int64
	Lifecycle   string
}

type AccountStatusSummary struct {
	Total             int64            `json:"total"`
	ByStatus          map[string]int64 `json:"by_status"`
	RateLimited       int64            `json:"rate_limited"`
	TempUnschedulable int64            `json:"temp_unschedulable"`
	Overloaded        int64            `json:"overloaded"`
	Paused            int64            `json:"paused"`
	ByPlatform        map[string]int64 `json:"by_platform"`
}
