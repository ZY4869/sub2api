package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func recordImageRouteRuntimeMetrics(normalize service.ProtocolNormalizeResult, statusCode int, upstreamStatusCode *int, durationMs int64) {
	if strings.TrimSpace(normalize.ImageRouteFamily) == "" {
		return
	}
	failureStatus := statusCode
	if upstreamStatusCode != nil && *upstreamStatusCode > 0 {
		failureStatus = *upstreamStatusCode
	}
	protocolruntime.RecordImageRoute(
		normalize.ImageRouteFamily,
		normalize.ImageResolvedProvider,
		normalize.ImageProtocolMode,
		normalize.ImageAction,
		normalize.ImageSizeTier,
		normalize.ImageCapabilityProfile,
		statusCode > 0 && statusCode < 400,
		durationMs,
		failureStatus,
	)
	if strings.TrimSpace(normalize.ImageRouteFamily) != service.PublicImageToolRouteFamily {
		return
	}
	if failureStatus >= 400 {
		protocolruntime.RecordResponsesImageToolFailure(normalize.ImageResolvedProvider)
	}
}
