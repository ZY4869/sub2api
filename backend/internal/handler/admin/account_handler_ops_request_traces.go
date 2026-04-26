package admin

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/gin-gonic/gin"
)

func recordProbeActionTrace(
	c *gin.Context,
	opsService *service.OpsService,
	requestID string,
	clientRequestID string,
	probeAction string,
	adminUserID *int64,
	accountID *int64,
	platform string,
	protocolIn string,
	protocolOut string,
	routePath string,
	requestedModel string,
	normalizedRequest map[string]any,
	gatewayResponse map[string]any,
	status string,
	statusCode int,
	duration time.Duration,
) {
	if c == nil || opsService == nil {
		return
	}

	normalizedRequestBytes, _ := json.Marshal(normalizedRequest)
	normalizedRequestJSON := string(normalizedRequestBytes)

	gatewayResponseBytes, _ := json.Marshal(gatewayResponse)
	gatewayResponseJSON := string(gatewayResponseBytes)

	trace := service.GatewayTraceContext{
		Normalize: service.ProtocolNormalizeResult{
			Platform:       strings.TrimSpace(platform),
			ProtocolIn:     strings.TrimSpace(protocolIn),
			ProtocolOut:    strings.TrimSpace(protocolOut),
			Channel:        "admin",
			RoutePath:      strings.TrimSpace(routePath),
			RequestType:    "probe_action",
			RequestedModel: strings.TrimSpace(requestedModel),
			ProbeAction:    strings.TrimSpace(probeAction),
		},
		NormalizedRequestJSON: &normalizedRequestJSON,
		GatewayResponseJSON:   &gatewayResponseJSON,
	}

	durationMs := duration.Milliseconds()
	recordedAt := time.Now().UTC()
	_ = opsService.RecordRequestTrace(c.Request.Context(), &service.OpsRecordRequestTraceInput{
		RequestID:       strings.TrimSpace(requestID),
		ClientRequestID: strings.TrimSpace(clientRequestID),
		UserID:          adminUserID,
		AccountID:       accountID,
		Status:          strings.TrimSpace(status),
		StatusCode:      statusCode,
		DurationMs:      durationMs,
		Trace:           trace,
		CreatedAt:       recordedAt,
	})
}

func redactPromptPreview(prompt string, limit int) string {
	prompt = strings.TrimSpace(prompt)
	if prompt == "" || limit <= 0 {
		return ""
	}
	redacted := logredact.RedactText(prompt, "sso_token")
	if len(redacted) <= limit {
		return redacted
	}
	return redacted[:limit] + "..."
}
