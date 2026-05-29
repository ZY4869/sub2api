package service

import (
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	"time"
)

const (
	opsRequestTraceDisabledMessage     = "ops request details is disabled"
	opsRequestTraceListPageSize        = 200
	opsRequestTraceExportMaxRows       = 50000
	opsRequestTraceRawExportMaxWindow  = 7 * 24 * time.Hour
	opsRequestTraceExportMaxWindow     = 30 * 24 * time.Hour
	opsRequestTraceInboundPreviewLimit = 512 * 1024
	opsRequestTraceRawRequestLimit     = 512 * 1024
	opsRequestTraceRawResponseLimit    = 1024 * 1024
	opsRequestTraceSearchTextLimit     = 4096
	opsRequestTracePayloadJSONLimit    = 64 * 1024
	opsRequestTraceDefaultSlowMs       = int64(60000)
)

var ErrOpsRequestTracesDisabled = infraerrors.NotFound("OPS_REQUEST_TRACES_DISABLED", opsRequestTraceDisabledMessage)

type OpsRecordRequestTraceInput struct {
	RequestID          string
	ClientRequestID    string
	UpstreamRequestID  string
	UserID             *int64
	APIKeyID           *int64
	AccountID          *int64
	GroupID            *int64
	Status             string
	StatusCode         int
	UpstreamStatusCode *int
	DurationMs         int64
	TTFTMs             *int64
	InputTokens        int
	OutputTokens       int
	TotalTokens        int
	Trace              GatewayTraceContext
	CreatedAt          time.Time
}

type opsRequestTraceRuntimeConfig struct {
	Enabled                  bool
	EncryptionKey            string
	RawAccessUserIDs         map[int64]struct{}
	RetentionDays            int
	PayloadPreviewLimitBytes int
	SuccessSampleRate        float64
	ForceCaptureSlowMs       int64
	RawExportMaxRows         int
}
