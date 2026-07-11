package service

import (
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type ImageBatchJobStatus string
type ImageBatchItemStatus string

const (
	ImageBatchJobCreated       ImageBatchJobStatus = "created"
	ImageBatchJobUploading     ImageBatchJobStatus = "uploading"
	ImageBatchJobSubmitted     ImageBatchJobStatus = "submitted"
	ImageBatchJobRunning       ImageBatchJobStatus = "running"
	ImageBatchJobIndexing      ImageBatchJobStatus = "indexing"
	ImageBatchJobSettling      ImageBatchJobStatus = "settling"
	ImageBatchJobCompleted     ImageBatchJobStatus = "completed"
	ImageBatchJobFailed        ImageBatchJobStatus = "failed"
	ImageBatchJobCancelled     ImageBatchJobStatus = "cancelled"
	ImageBatchJobOutputDeleted ImageBatchJobStatus = "output_deleted"

	ImageBatchItemPending   ImageBatchItemStatus = "pending"
	ImageBatchItemSuccess   ImageBatchItemStatus = "success"
	ImageBatchItemFailed    ImageBatchItemStatus = "failed"
	ImageBatchItemCancelled ImageBatchItemStatus = "cancelled"
)

const (
	ImageBatchProviderGeminiAPI = "gemini_api"
	ImageBatchProviderVertex    = "vertex"
	ImageBatchProviderGemini    = ImageBatchProviderGeminiAPI
)

var (
	ErrImageBatchUnavailable      = infraerrors.Forbidden("IMAGE_BATCH_DISABLED", "image batch generation is not enabled for this API key")
	ErrImageBatchInvalidRequest   = infraerrors.BadRequest("IMAGE_BATCH_INVALID_REQUEST", "invalid image batch request")
	ErrImageBatchInvalidModel     = infraerrors.BadRequest("IMAGE_BATCH_MODEL_NOT_ALLOWED", "selected image batch model is not available")
	ErrImageBatchInvalidProvider  = infraerrors.BadRequest("IMAGE_BATCH_PROVIDER_NOT_ALLOWED", "selected image batch provider is not available")
	ErrImageBatchNotFound         = infraerrors.NotFound("IMAGE_BATCH_NOT_FOUND", "image batch job not found")
	ErrImageBatchConflict         = infraerrors.Conflict("IMAGE_BATCH_STATE_CONFLICT", "image batch job is not in a valid state for this operation")
	ErrImageBatchOutputNotFound   = infraerrors.NotFound("IMAGE_BATCH_OUTPUT_NOT_FOUND", "image batch output not found")
	ErrImageBatchDownloadTooLarge = infraerrors.BadRequest("IMAGE_BATCH_DOWNLOAD_TOO_LARGE", "image batch download exceeds the configured limit")
	ErrImageBatchDownloadBusy     = infraerrors.TooManyRequests("IMAGE_BATCH_DOWNLOAD_BUSY", "too many image batch downloads are running; please retry shortly")
)

type ImageBatchSubmitRequest struct {
	Provider string                 `json:"provider"`
	Model    string                 `json:"model"`
	Size     string                 `json:"size"`
	Items    []ImageBatchSubmitItem `json:"items"`
	Metadata map[string]any         `json:"metadata,omitempty"`
}

type ImageBatchSubmitItem struct {
	CustomID string         `json:"custom_id"`
	Prompt   string         `json:"prompt"`
	N        int            `json:"n"`
	Size     string         `json:"size"`
	Metadata map[string]any `json:"metadata,omitempty"`
}

type ImageBatchJob struct {
	ID                     string
	UserID                 int64
	APIKeyID               int64
	GroupID                *int64
	Provider               string
	DisplayModelID         string
	TargetModelID          string
	Size                   string
	Status                 ImageBatchJobStatus
	ItemCount              int
	SuccessCount           int
	FailedCount            int
	CancelledCount         int
	HoldRequestID          string
	IdempotencyKeyHash     string
	RequestFingerprint     string
	ProviderBatchName      string
	ProviderResultFileName string
	FriendlyError          string
	ErrorID                string
	Metadata               map[string]any
	SubmittedAt            *time.Time
	StartedAt              *time.Time
	CompletedAt            *time.Time
	CancelledAt            *time.Time
	OutputsDeletedAt       *time.Time
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

type ImageBatchItem struct {
	ID            int64
	JobID         string
	CustomID      string
	Prompt        string
	N             int
	Size          string
	Status        ImageBatchItemStatus
	OutputCount   int
	FriendlyError string
	ErrorID       string
	Metadata      map[string]any
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type ImageBatchOutput struct {
	ID             int64
	JobID          string
	ItemID         int64
	CustomID       string
	ContentType    string
	StorageBackend string
	Content        []byte
	SizeBytes      int64
	SHA256         string
	Metadata       map[string]any
	CreatedAt      time.Time
}

type ImageBatchGroupSettings struct {
	Enabled             bool
	AllowedProviders    []string
	AllowedModels       []string
	MaxItems            int
	MaxDownloadBytes    int64
	DownloadConcurrency int
}

type ImageBatchJobResponse struct {
	ID             string              `json:"id"`
	Object         string              `json:"object"`
	Status         ImageBatchJobStatus `json:"status"`
	Provider       string              `json:"provider"`
	DisplayModelID string              `json:"display_model_id"`
	Size           string              `json:"size,omitempty"`
	Counts         ImageBatchJobCounts `json:"counts"`
	FriendlyError  string              `json:"friendly_error,omitempty"`
	ErrorID        string              `json:"error_id,omitempty"`
	CreatedAt      time.Time           `json:"created_at"`
	UpdatedAt      time.Time           `json:"updated_at"`
	SubmittedAt    *time.Time          `json:"submitted_at,omitempty"`
	CompletedAt    *time.Time          `json:"completed_at,omitempty"`
	CancelledAt    *time.Time          `json:"cancelled_at,omitempty"`
}

type ImageBatchJobCounts struct {
	Items     int `json:"items"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Cancelled int `json:"cancelled"`
}

type ImageBatchItemResponse struct {
	CustomID      string               `json:"custom_id"`
	Status        ImageBatchItemStatus `json:"status"`
	OutputCount   int                  `json:"output_count"`
	FriendlyError string               `json:"friendly_error,omitempty"`
	ErrorID       string               `json:"error_id,omitempty"`
	Metadata      map[string]any       `json:"metadata,omitempty"`
	CreatedAt     time.Time            `json:"created_at"`
	UpdatedAt     time.Time            `json:"updated_at"`
}

type ImageBatchModelResponse struct {
	ID          string `json:"id"`
	Object      string `json:"object"`
	DisplayName string `json:"display_name,omitempty"`
	Provider    string `json:"provider"`
}
