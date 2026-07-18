package securityaudit

import "time"

type Job struct {
	ID int64
	Request
	PromptHash        string
	RedactedPreview   string
	PromptLength      int
	MessageCount      int
	ExecutionMode     string
	ConfigVersion     int64
	Status            string
	Attempts          int
	MaxAttempts       int
	NextAttemptAt     time.Time
	ProcessingStarted *time.Time
	ProcessedAt       *time.Time
	LastErrorCode     string
	LastErrorMessage  string
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Event struct {
	ID              int64              `json:"id"`
	JobID           int64              `json:"job_id"`
	RequestID       string             `json:"request_id"`
	UserID          *int64             `json:"user_id,omitempty"`
	Username        string             `json:"username_snapshot"`
	UserEmail       string             `json:"user_email_snapshot"`
	APIKeyID        *int64             `json:"api_key_id,omitempty"`
	APIKeyName      string             `json:"api_key_name_snapshot"`
	GroupID         *int64             `json:"group_id,omitempty"`
	GroupName       string             `json:"group_name"`
	Provider        string             `json:"provider"`
	Endpoint        string             `json:"endpoint"`
	Protocol        string             `json:"protocol"`
	Model           string             `json:"model"`
	PromptHash      string             `json:"prompt_hash"`
	RedactedPreview string             `json:"redacted_preview"`
	FullPrompt      string             `json:"full_prompt,omitempty"`
	Stage           string             `json:"stage"`
	Decision        EventDecision      `json:"decision"`
	RiskLevel       RiskLevel          `json:"risk_level"`
	Action          Action             `json:"action"`
	Categories      []string           `json:"categories"`
	MatchedScanners []string           `json:"matched_scanners"`
	ScannerScores   map[string]float64 `json:"scanner_scores"`
	ScannerEvidence map[string]string  `json:"scanner_evidence"`
	ScannerBackend  string             `json:"scanner_backend"`
	ScannerVersion  string             `json:"scanner_version"`
	GuardEndpointID string             `json:"guard_endpoint_id"`
	PolicyID        string             `json:"policy_id"`
	PolicyVersion   int                `json:"policy_version"`
	ConfigVersion   int64              `json:"config_version"`
	ChunkTotal      int                `json:"chunk_total"`
	LatencyMS       int                `json:"latency_ms"`
	IssueSummaries  []IssueSummary     `json:"issue_summaries,omitempty"`
	Metadata        map[string]any     `json:"metadata,omitempty"`
	CreatedAt       time.Time          `json:"created_at"`
}

type EventFilter struct {
	Page            int
	PageSize        int
	Decision        string
	RiskLevel       string
	Endpoint        string
	UserID          *int64
	APIKeyID        *int64
	GroupID         *int64
	PromptHash      string
	RequestID       string
	Protocol        string
	Keyword         string
	StartAt         *time.Time
	EndAt           *time.Time
	IncludeFullText bool
}

type EventList struct {
	Items    []*Event `json:"items"`
	Total    int64    `json:"total"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

type DeletePreview struct {
	Matched           int64     `json:"matched"`
	SnapshotMaxID     int64     `json:"snapshot_max_id"`
	FilterHash        string    `json:"filter_hash"`
	ConfirmationToken string    `json:"confirmation_token,omitempty"`
	ExpiresAt         time.Time `json:"expires_at,omitempty"`
}

type DeleteResult struct {
	Deleted int64   `json:"deleted"`
	JobIDs  []int64 `json:"-"`
}

type IssueSummary struct {
	Code         string `json:"code"`
	Title        string `json:"title"`
	Description  string `json:"description"`
	EvidenceHash string `json:"evidence_hash"`
}
