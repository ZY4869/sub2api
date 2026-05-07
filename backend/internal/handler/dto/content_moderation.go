package dto

type ContentModerationAudit struct {
	ID              int64  `json:"id"`
	RequestID       string `json:"request_id"`
	ClientRequestID string `json:"client_request_id"`
	UserID          *int64 `json:"user_id"`
	APIKeyID        *int64 `json:"api_key_id"`
	Provider        string `json:"provider"`
	Model           string `json:"model"`
	SourceEndpoint  string `json:"source_endpoint"`
	ContentHash     string `json:"content_hash"`
	ContentSummary  string `json:"content_summary"`
	Hit             bool   `json:"hit"`
	DedupeHit       bool   `json:"dedupe_hit"`
	ErrorReason     string `json:"error_reason"`
	LatencyMs       int    `json:"latency_ms"`
	CreatedAt       string `json:"created_at"`
}
