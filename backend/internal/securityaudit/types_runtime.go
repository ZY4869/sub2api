package securityaudit

import "time"

type RuntimeSnapshot struct {
	ProcessStatus         string                 `json:"process_status"`
	EffectiveMode         Mode                   `json:"effective_mode"`
	ExpectedConfigVersion int64                  `json:"expected_config_version"`
	ActiveConfigVersion   int64                  `json:"active_config_version"`
	ConfigLoadedAt        *time.Time             `json:"config_loaded_at,omitempty"`
	ConfigLoadError       string                 `json:"config_load_error,omitempty"`
	WorkerTotal           int                    `json:"worker_total"`
	WorkerActive          int64                  `json:"worker_active"`
	QueueCapacity         int                    `json:"queue_capacity"`
	Queue                 QueueStats             `json:"queue"`
	DatabaseStatus        string                 `json:"database_status"`
	RedisStatus           string                 `json:"redis_status"`
	Endpoints             map[string]ProbeResult `json:"endpoints"`
	GuardMetrics          MetricsSnapshot        `json:"guard_metrics"`
	EnqueuedTotal         int64                  `json:"enqueued_total"`
	DroppedTotal          int64                  `json:"dropped_total"`
	ProcessedTotal        int64                  `json:"processed_total"`
	FailedTotal           int64                  `json:"failed_total"`
	WorkerHeartbeatAt     *time.Time             `json:"worker_heartbeat_at,omitempty"`
	LastProcessedAt       *time.Time             `json:"last_processed_at,omitempty"`
	LastErrorCode         string                 `json:"last_error_code,omitempty"`
	LastErrorMessage      string                 `json:"last_error_message,omitempty"`
}

type QueueStats struct {
	Queued     int64 `json:"queued"`
	Processing int64 `json:"processing"`
	Retry      int64 `json:"retry"`
	Done       int64 `json:"done"`
	Failed     int64 `json:"failed"`
}

type ProbeResult struct {
	OK           bool      `json:"ok"`
	Status       string    `json:"status"`
	Message      string    `json:"message"`
	ErrorCode    string    `json:"error_code,omitempty"`
	HTTPStatus   int       `json:"http_status,omitempty"`
	Retryable    bool      `json:"retryable"`
	TokenApplied bool      `json:"token_applied"`
	LatencyMS    int       `json:"latency_ms"`
	CheckedAt    time.Time `json:"checked_at"`
}

type MetricsSnapshot struct {
	Allow       int64 `json:"allow"`
	Flag        int64 `json:"flag"`
	Block       int64 `json:"block"`
	Unavailable int64 `json:"unavailable"`
	Timeout     int64 `json:"timeout"`
	Failover    int64 `json:"failover"`
	Bulkhead    int64 `json:"bulkhead"`
	Enqueued    int64 `json:"enqueued"`
	Dropped     int64 `json:"dropped"`
	Processed   int64 `json:"processed"`
	Failed      int64 `json:"failed"`
}
