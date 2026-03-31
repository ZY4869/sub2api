package service

import (
	"context"
	"time"
)

const (
	SettingKeyGoogleBatchArchiveSettings = "google_batch_archive_settings"

	GoogleBatchArchivePublicProtocolAIStudio = "ai_studio"

	GoogleBatchArchiveConversionNone             = "none"
	GoogleBatchArchiveConversionAIStudioToVertex = "aistudio_to_vertex"

	GoogleBatchArchiveStateCreated   = "created"
	GoogleBatchArchiveStateRunning   = "running"
	GoogleBatchArchiveStateSucceeded = "succeeded"
	GoogleBatchArchiveStateFailed    = "failed"
	GoogleBatchArchiveStateCancelled = "cancelled"
	GoogleBatchArchiveStateDeleting  = "deleting"
	GoogleBatchArchiveStateUnknown   = "unknown"

	GoogleBatchArchiveLifecyclePending  = "pending"
	GoogleBatchArchiveLifecycleArchived = "archived"
	GoogleBatchArchiveLifecycleFailed   = "failed"
	GoogleBatchArchiveLifecycleDeleted  = "deleted"

	GoogleBatchArchiveBillingPending = "pending"
	GoogleBatchArchiveBillingSettled = "settled"
	GoogleBatchArchiveBillingSkipped = "skipped"

	GoogleBatchArchiveStorageBackendLocalFS = "local_fs"

	GoogleBatchArchiveBillingModeLogOnly       = "log_only"
	GoogleBatchArchiveBillingModeArchiveCharge = "archive_charge"

	GoogleBatchArchiveResourceKindBatch = "batch"
	GoogleBatchArchiveResourceKindFile  = "file"

	UsageOperationBatchCreate            = "batch_create"
	UsageOperationBatchSettlement        = "batch_settlement"
	UsageOperationBatchStatus            = "batch_status"
	UsageOperationGetFileMetadata        = "get_file_metadata"
	UsageOperationOfficialResultDownload = "official_result_download"
	UsageOperationLocalArchiveDownload   = "local_archive_download"

	UsageChargeSourceModelBatch      = "model_batch"
	UsageChargeSourceArchiveDownload = "archive_download"
	UsageChargeSourceNone            = "none"
)

const (
	googleBatchArchiveDefaultPollMinIntervalSeconds = 300
	googleBatchArchiveDefaultPollMaxIntervalSeconds = 1800
	googleBatchArchiveDefaultPollBackoffFactor      = 2
	googleBatchArchiveDefaultPollJitterSeconds      = 30
	googleBatchArchiveDefaultPollMaxConcurrency     = 2
	googleBatchArchiveDefaultPrefetchAfterHours     = 40
	googleBatchArchiveDefaultDownloadTimeoutSeconds = 180
	googleBatchArchiveDefaultCleanupIntervalMinutes = 60
	googleBatchArchiveDefaultLocalStorageRoot       = "/app/data/google-batch"
	googleBatchArchiveDefaultRetentionDays          = 7
)

type GoogleBatchArchiveSettings struct {
	Enabled                bool   `json:"enabled"`
	PollMinIntervalSeconds int    `json:"poll_min_interval_seconds"`
	PollMaxIntervalSeconds int    `json:"poll_max_interval_seconds"`
	PollBackoffFactor      int    `json:"poll_backoff_factor"`
	PollJitterSeconds      int    `json:"poll_jitter_seconds"`
	PollMaxConcurrency     int    `json:"poll_max_concurrency"`
	PrefetchAfterHours     int    `json:"prefetch_after_hours"`
	DownloadTimeoutSeconds int    `json:"download_timeout_seconds"`
	CleanupIntervalMinutes int    `json:"cleanup_interval_minutes"`
	LocalStorageRoot       string `json:"local_storage_root"`
}

func DefaultGoogleBatchArchiveSettings() *GoogleBatchArchiveSettings {
	return &GoogleBatchArchiveSettings{
		Enabled:                false,
		PollMinIntervalSeconds: googleBatchArchiveDefaultPollMinIntervalSeconds,
		PollMaxIntervalSeconds: googleBatchArchiveDefaultPollMaxIntervalSeconds,
		PollBackoffFactor:      googleBatchArchiveDefaultPollBackoffFactor,
		PollJitterSeconds:      googleBatchArchiveDefaultPollJitterSeconds,
		PollMaxConcurrency:     googleBatchArchiveDefaultPollMaxConcurrency,
		PrefetchAfterHours:     googleBatchArchiveDefaultPrefetchAfterHours,
		DownloadTimeoutSeconds: googleBatchArchiveDefaultDownloadTimeoutSeconds,
		CleanupIntervalMinutes: googleBatchArchiveDefaultCleanupIntervalMinutes,
		LocalStorageRoot:       googleBatchArchiveDefaultLocalStorageRoot,
	}
}

func NormalizeGoogleBatchArchiveSettings(settings *GoogleBatchArchiveSettings) *GoogleBatchArchiveSettings {
	base := DefaultGoogleBatchArchiveSettings()
	if settings == nil {
		return base
	}
	if settings.PollMinIntervalSeconds > 0 {
		base.PollMinIntervalSeconds = settings.PollMinIntervalSeconds
	}
	if settings.PollMaxIntervalSeconds > 0 {
		base.PollMaxIntervalSeconds = settings.PollMaxIntervalSeconds
	}
	if base.PollMaxIntervalSeconds < base.PollMinIntervalSeconds {
		base.PollMaxIntervalSeconds = base.PollMinIntervalSeconds
	}
	if settings.PollBackoffFactor > 0 {
		base.PollBackoffFactor = settings.PollBackoffFactor
	}
	if settings.PollJitterSeconds >= 0 {
		base.PollJitterSeconds = settings.PollJitterSeconds
	}
	if settings.PollMaxConcurrency > 0 {
		base.PollMaxConcurrency = settings.PollMaxConcurrency
	}
	if settings.PrefetchAfterHours > 0 {
		base.PrefetchAfterHours = settings.PrefetchAfterHours
	}
	if settings.DownloadTimeoutSeconds > 0 {
		base.DownloadTimeoutSeconds = settings.DownloadTimeoutSeconds
	}
	if settings.CleanupIntervalMinutes > 0 {
		base.CleanupIntervalMinutes = settings.CleanupIntervalMinutes
	}
	if trimmed := normalizeFileStorageRoot(settings.LocalStorageRoot); trimmed != "" {
		base.LocalStorageRoot = trimmed
	}
	base.Enabled = settings.Enabled
	return base
}

type GoogleBatchArchiveJob struct {
	ID                       int64
	PublicBatchName          string
	PublicProtocol           string
	ExecutionProviderFamily  string
	ExecutionBatchName       string
	SourceAccountID          int64
	ExecutionAccountID       int64
	APIKeyID                 *int64
	GroupID                  *int64
	UserID                   *int64
	RequestedModel           string
	ConversionDirection      string
	State                    string
	OfficialExpiresAt        *time.Time
	PrefetchDueAt            *time.Time
	LastPublicResultAccessAt *time.Time
	NextPollAt               *time.Time
	PollAttempts             int
	ArchiveState             string
	BillingSettlementState   string
	RetentionExpiresAt       *time.Time
	MetadataJSON             map[string]any
	CreatedAt                time.Time
	UpdatedAt                time.Time
	DeletedAt                *time.Time
}

type GoogleBatchArchiveObject struct {
	ID                    int64
	JobID                 int64
	PublicResourceKind    string
	PublicResourceName    string
	ExecutionResourceName string
	StorageBackend        string
	RelativePath          string
	ContentType           string
	SizeBytes             int64
	SHA256                string
	IsResultPayload       bool
	MetadataJSON          map[string]any
	CreatedAt             time.Time
	UpdatedAt             time.Time
	DeletedAt             *time.Time
}

type GoogleBatchArchiveJobRepository interface {
	Upsert(ctx context.Context, job *GoogleBatchArchiveJob) error
	GetByID(ctx context.Context, id int64) (*GoogleBatchArchiveJob, error)
	GetByPublicBatchName(ctx context.Context, publicBatchName string) (*GoogleBatchArchiveJob, error)
	GetByExecutionBatchName(ctx context.Context, executionBatchName string) (*GoogleBatchArchiveJob, error)
	ListDueForPoll(ctx context.Context, before time.Time, limit int) ([]*GoogleBatchArchiveJob, error)
	ListDueForPrefetch(ctx context.Context, before time.Time, limit int) ([]*GoogleBatchArchiveJob, error)
	ListExpiredForCleanup(ctx context.Context, before time.Time, limit int) ([]*GoogleBatchArchiveJob, error)
	TouchLastPublicResultAccess(ctx context.Context, id int64, accessedAt time.Time) error
	SoftDelete(ctx context.Context, id int64) error
}

type GoogleBatchArchiveObjectRepository interface {
	Upsert(ctx context.Context, object *GoogleBatchArchiveObject) error
	GetByPublicResource(ctx context.Context, publicResourceKind string, publicResourceName string) (*GoogleBatchArchiveObject, error)
	ListByJobID(ctx context.Context, jobID int64) ([]*GoogleBatchArchiveObject, error)
	SoftDeleteByJobID(ctx context.Context, jobID int64) error
}
