package service

import (
	"context"
	"strings"
	"time"
)

const (
	DocumentAIProviderBaidu = PlatformBaiduDocumentAI

	DocumentAIModelPPOCRV5Server = "pp-ocrv5-server"
	DocumentAIModelPPStructureV3 = "pp-structurev3"
	DocumentAIModelPaddleOCRVL   = "paddleocr-vl"
	DocumentAIModelPaddleOCRVL15 = "paddleocr-vl-1.5"

	DocumentAIJobModeAsync  = "async"
	DocumentAIJobModeDirect = "direct"

	DocumentAIJobStatusPending   = "pending"
	DocumentAIJobStatusRunning   = "running"
	DocumentAIJobStatusSucceeded = "succeeded"
	DocumentAIJobStatusFailed    = "failed"
	DocumentAIJobStatusCanceled  = "canceled"

	DocumentAISourceTypeFile       = "file"
	DocumentAISourceTypeFileURL    = "file_url"
	DocumentAISourceTypeFileBase64 = "file_base64"

	DocumentAIFileTypeImage = "image"
	DocumentAIFileTypePDF   = "pdf"
)

type DocumentAIModelDescriptor struct {
	ID                 string   `json:"id"`
	DisplayName        string   `json:"display_name"`
	Provider           string   `json:"provider"`
	Modes              []string `json:"modes"`
	SupportsMultipart  bool     `json:"supports_multipart"`
	SupportsFileURL    bool     `json:"supports_file_url"`
	SupportsFileBase64 bool     `json:"supports_file_base64"`
}

type DocumentAIJob struct {
	ID                   int64
	JobID                string
	ProviderJobID        *string
	ProviderBatchID      *string
	AccountID            *int64
	UserID               int64
	APIKeyID             int64
	GroupID              *int64
	Mode                 string
	Model                string
	SourceType           string
	FileName             *string
	ContentType          *string
	FileSize             *int64
	FileHash             *string
	Status               string
	ProviderResultJSON   *string
	NormalizedResultJSON *string
	ErrorCode            *string
	ErrorMessage         *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	CompletedAt          *time.Time
	LastPolledAt         *time.Time
}

type DocumentAISubmitJobInput struct {
	APIKey      *APIKey
	GroupID     int64
	Model       string
	SourceType  string
	FileName    string
	ContentType string
	FileSize    int64
	FileHash    string
	FileBytes   []byte
	FileURL     string
	Options     map[string]any
}

type DocumentAIParseDirectInput struct {
	APIKey      *APIKey
	GroupID     int64
	Model       string
	SourceType  string
	FileType    string
	FileName    string
	ContentType string
	FileSize    int64
	FileHash    string
	FileBytes   []byte
	FileBase64  string
	Options     map[string]any
}

type DocumentAIResultEnvelope struct {
	Provider      string  `json:"provider"`
	Mode          string  `json:"mode"`
	Model         string  `json:"model"`
	Status        string  `json:"status"`
	ProviderJobID *string `json:"provider_job_id,omitempty"`
	Text          string  `json:"text"`
	PageCount     int     `json:"page_count"`
	TableCount    int     `json:"table_count"`
	HasLayout     bool    `json:"has_layout"`
}

type DocumentAIJobRepository interface {
	Create(ctx context.Context, job *DocumentAIJob) error
	GetByJobIDForUser(ctx context.Context, jobID string, userID int64) (*DocumentAIJob, error)
	UpdateAfterSubmit(ctx context.Context, jobID string, providerJobID, providerBatchID *string, status string, providerResultJSON *string) error
	ListPollable(ctx context.Context, limit int) ([]DocumentAIJob, error)
	MarkRunning(ctx context.Context, jobID string, providerResultJSON *string) error
	MarkSucceeded(ctx context.Context, jobID string, providerResultJSON, normalizedResultJSON *string) error
	MarkFailed(ctx context.Context, jobID string, providerResultJSON *string, errorCode, errorMessage string) error
	TouchLastPolledAt(ctx context.Context, jobID string) error
}

func BuiltinDocumentAIModels() []DocumentAIModelDescriptor {
	return []DocumentAIModelDescriptor{
		{
			ID:                 DocumentAIModelPPOCRV5Server,
			DisplayName:        "PP-OCRv5 Server",
			Provider:           DocumentAIProviderBaidu,
			Modes:              []string{DocumentAIJobModeAsync, DocumentAIJobModeDirect},
			SupportsMultipart:  true,
			SupportsFileURL:    true,
			SupportsFileBase64: true,
		},
		{
			ID:                 DocumentAIModelPPStructureV3,
			DisplayName:        "PP-StructureV3",
			Provider:           DocumentAIProviderBaidu,
			Modes:              []string{DocumentAIJobModeAsync},
			SupportsMultipart:  true,
			SupportsFileURL:    true,
			SupportsFileBase64: false,
		},
		{
			ID:                 DocumentAIModelPaddleOCRVL,
			DisplayName:        "PaddleOCR-VL",
			Provider:           DocumentAIProviderBaidu,
			Modes:              []string{DocumentAIJobModeAsync},
			SupportsMultipart:  true,
			SupportsFileURL:    true,
			SupportsFileBase64: false,
		},
		{
			ID:                 DocumentAIModelPaddleOCRVL15,
			DisplayName:        "PaddleOCR-VL 1.5",
			Provider:           DocumentAIProviderBaidu,
			Modes:              []string{DocumentAIJobModeAsync, DocumentAIJobModeDirect},
			SupportsMultipart:  true,
			SupportsFileURL:    true,
			SupportsFileBase64: true,
		},
	}
}

func DocumentAIModelSupportsDirect(model string) bool {
	switch normalizeDocumentAIModelID(model) {
	case DocumentAIModelPPOCRV5Server, DocumentAIModelPaddleOCRVL15:
		return true
	default:
		return false
	}
}

func DocumentAIModelSupportsAsync(model string) bool {
	switch normalizeDocumentAIModelID(model) {
	case DocumentAIModelPPOCRV5Server, DocumentAIModelPPStructureV3, DocumentAIModelPaddleOCRVL, DocumentAIModelPaddleOCRVL15:
		return true
	default:
		return false
	}
}

func normalizeDocumentAIModelID(model string) string {
	switch trimLower(model) {
	case "pp-ocrv5-server", "pp-ocrv5", "ppocrv5", "ppocrv5-server":
		return DocumentAIModelPPOCRV5Server
	case "pp-structurev3", "pp-structure-v3":
		return DocumentAIModelPPStructureV3
	case "paddleocr-vl":
		return DocumentAIModelPaddleOCRVL
	case "paddleocr-vl-1.5", "paddleocr-vl-15":
		return DocumentAIModelPaddleOCRVL15
	default:
		return ""
	}
}

func documentAIProviderModelID(model string) string {
	switch normalizeDocumentAIModelID(model) {
	case DocumentAIModelPPOCRV5Server:
		return "PP-OCRv5"
	case DocumentAIModelPPStructureV3:
		return "PP-StructureV3"
	case DocumentAIModelPaddleOCRVL:
		return "PaddleOCR-VL"
	case DocumentAIModelPaddleOCRVL15:
		return "PaddleOCR-VL-1.5"
	default:
		return ""
	}
}

func trimLower(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
