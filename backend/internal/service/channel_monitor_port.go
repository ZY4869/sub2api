package service

import (
	"context"
	"time"
)

const (
	ChannelMonitorProviderOpenAI      = "openai"
	ChannelMonitorProviderAnthropic   = "anthropic"
	ChannelMonitorProviderGemini      = "gemini"
	ChannelMonitorProviderGrok        = "grok"
	ChannelMonitorProviderAntigravity = "antigravity"
)

const (
	ChannelMonitorBodyOverrideModeOff     = "off"
	ChannelMonitorBodyOverrideModeMerge   = "merge"
	ChannelMonitorBodyOverrideModeReplace = "replace"
)

const (
	ChannelMonitorStatusSuccess  = "success"
	ChannelMonitorStatusDegraded = "degraded"
	ChannelMonitorStatusFailure  = "failure"
)

type ChannelMonitor struct {
	ID                 int64    `json:"id"`
	Name               string   `json:"name"`
	Provider           string   `json:"provider"`
	Endpoint           string   `json:"endpoint"`
	APIKeyEncrypted    *string  `json:"-"`
	IntervalSeconds    int      `json:"interval_seconds"`
	Enabled            bool     `json:"enabled"`
	PrimaryModelID     string   `json:"primary_model_id"`
	AdditionalModelIDs []string `json:"additional_model_ids"`

	TemplateID       *int64            `json:"template_id,omitempty"`
	ExtraHeaders     map[string]string `json:"extra_headers"`
	BodyOverrideMode string            `json:"body_override_mode"`
	BodyOverride     map[string]any    `json:"body_override"`

	LastRunAt *time.Time `json:"last_run_at,omitempty"`
	NextRunAt *time.Time `json:"next_run_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

type ChannelMonitorHistory struct {
	ID           int64     `json:"id"`
	MonitorID    int64     `json:"monitor_id"`
	ModelID      string    `json:"model_id"`
	Status       string    `json:"status"`
	ResponseText string    `json:"response_text"`
	ErrorMessage string    `json:"error_message"`
	HTTPStatus   *int      `json:"http_status,omitempty"`
	LatencyMs    int64     `json:"latency_ms"`
	StartedAt    time.Time `json:"started_at"`
	FinishedAt   time.Time `json:"finished_at"`
	CreatedAt    time.Time `json:"created_at"`
}

type ChannelMonitorRequestTemplate struct {
	ID               int64             `json:"id"`
	Name             string            `json:"name"`
	Provider         string            `json:"provider"`
	Description      *string           `json:"description,omitempty"`
	ExtraHeaders     map[string]string `json:"extra_headers"`
	BodyOverrideMode string            `json:"body_override_mode"`
	BodyOverride     map[string]any    `json:"body_override"`
	CreatedAt        time.Time         `json:"created_at"`
	UpdatedAt        time.Time         `json:"updated_at"`
}

type ChannelMonitorDailyRollup struct {
	MonitorID       int64
	ModelID         string
	Day             time.Time
	TotalChecks     int64
	AvailableChecks int64
	DegradedChecks  int64
	TotalLatencyMs  int64
	MaxLatencyMs    int64
}

type ChannelMonitorAvailabilityCounts struct {
	TotalChecks     int64
	AvailableChecks int64
	DegradedChecks  int64
}

type ChannelMonitorAvailabilityWindows struct {
	Last7  ChannelMonitorAvailabilityCounts
	Last15 ChannelMonitorAvailabilityCounts
	Last30 ChannelMonitorAvailabilityCounts
}

type ChannelMonitorRepository interface {
	Create(ctx context.Context, monitor *ChannelMonitor) (*ChannelMonitor, error)
	GetByID(ctx context.Context, id int64) (*ChannelMonitor, error)
	ListAll(ctx context.Context) ([]*ChannelMonitor, error)
	ListEnabled(ctx context.Context) ([]*ChannelMonitor, error)
	ClaimDue(ctx context.Context, now time.Time, limit int) ([]*ChannelMonitor, error)
	Update(ctx context.Context, monitor *ChannelMonitor) (*ChannelMonitor, error)
	Delete(ctx context.Context, id int64) error
}

type ChannelMonitorHistoryRepository interface {
	Create(ctx context.Context, history *ChannelMonitorHistory) (*ChannelMonitorHistory, error)
	ListByMonitorID(ctx context.Context, monitorID int64, limit int) ([]*ChannelMonitorHistory, error)
	ListLatestByMonitorIDs(ctx context.Context, monitorIDs []int64) ([]*ChannelMonitorHistory, error)
	ListPrimaryTimelineByMonitorIDs(ctx context.Context, monitorIDs []int64, limitPerMonitor int) ([]*ChannelMonitorHistory, error)
	ListLatestByMonitorID(ctx context.Context, monitorID int64) ([]*ChannelMonitorHistory, error)
	ListForAggregation(ctx context.Context, afterID int64, limit int) ([]*ChannelMonitorHistory, error)
	PruneBefore(ctx context.Context, before time.Time) (int64, error)
}

type ChannelMonitorRollupRepository interface {
	UpsertIncrement(ctx context.Context, monitorID int64, modelID string, day time.Time, deltaTotal int64, deltaAvailable int64, deltaDegraded int64, deltaLatency int64, maxLatencyCandidate int64) error
	SumAvailability(ctx context.Context, monitorIDs []int64, startDay time.Time) (map[int64]map[string]*ChannelMonitorDailyRollup, error)
	SumAvailabilityWindows(ctx context.Context, monitorID int64, start7 time.Time, start15 time.Time, start30 time.Time) (map[string]*ChannelMonitorAvailabilityWindows, error)
	PruneBeforeDay(ctx context.Context, beforeDay time.Time) (int64, error)
}

type ChannelMonitorAggregationRepository interface {
	GetWatermark(ctx context.Context) (int64, error)
	SetWatermark(ctx context.Context, lastHistoryID int64) error
}

type ChannelMonitorTemplateRepository interface {
	Create(ctx context.Context, tpl *ChannelMonitorRequestTemplate) (*ChannelMonitorRequestTemplate, error)
	GetByID(ctx context.Context, id int64) (*ChannelMonitorRequestTemplate, error)
	ListAll(ctx context.Context) ([]*ChannelMonitorRequestTemplate, error)
	Update(ctx context.Context, tpl *ChannelMonitorRequestTemplate) (*ChannelMonitorRequestTemplate, error)
	Delete(ctx context.Context, id int64) error
	ListAssociatedMonitors(ctx context.Context, templateID int64) ([]*ChannelMonitor, error)
}
