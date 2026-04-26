package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/util/logredact"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	accountTestOpsTestRunIDContextKey        = "account_test_ops_test_run_id"
	accountTestOpsProbeActionBaseContextKey  = "account_test_ops_probe_action_base"
	accountTestOpsAdminUserIDContextKey      = "account_test_ops_admin_user_id"
	accountTestOpsCollectorContextKey        = "account_test_ops_collector"
	accountTestOpsResponsePreviewLimitBytes  = 8 * 1024
	accountTestOpsUpstreamBodyPreviewMaxByte = 8 * 1024
)

type accountTestOpsCollector struct {
	TestRunID       string
	ProbeActionBase string
	AdminUserID     *int64
	StartedAt       time.Time

	RuntimeMeta      accountTestRuntimeMeta
	RequestedModelID string
	ResolvedTestMode string
	UpstreamStatus   *int
	UpstreamBodyText string
	ResponsePreview  string
	PreviewTruncated bool
	ErrorMessage     string
}

func (s *AccountTestService) SetOpsService(opsService *OpsService) {
	s.opsService = opsService
}

func readStringContextValue(c *gin.Context, key string) string {
	if c == nil {
		return ""
	}
	value, ok := c.Get(key)
	if !ok {
		return ""
	}
	if v, ok := value.(string); ok {
		return strings.TrimSpace(v)
	}
	return strings.TrimSpace(fmt.Sprint(value))
}

func readInt64ContextValue(c *gin.Context, key string) *int64 {
	if c == nil {
		return nil
	}
	value, ok := c.Get(key)
	if !ok {
		return nil
	}
	if v, ok := value.(int64); ok {
		return &v
	}
	if v, ok := value.(*int64); ok {
		return v
	}
	return nil
}

func (s *AccountTestService) ensureOpsCollector(c *gin.Context) *accountTestOpsCollector {
	if c == nil {
		return nil
	}

	if existing, ok := c.Get(accountTestOpsCollectorContextKey); ok {
		if collector, ok := existing.(*accountTestOpsCollector); ok && collector != nil {
			if collector.StartedAt.IsZero() {
				collector.StartedAt = time.Now()
			}
			return collector
		}
	}

	testRunID := readStringContextValue(c, accountTestOpsTestRunIDContextKey)
	probeActionBase := readStringContextValue(c, accountTestOpsProbeActionBaseContextKey)
	if testRunID == "" || probeActionBase == "" {
		return nil
	}

	collector := &accountTestOpsCollector{
		TestRunID:       testRunID,
		ProbeActionBase: probeActionBase,
		AdminUserID:     readInt64ContextValue(c, accountTestOpsAdminUserIDContextKey),
		StartedAt:       time.Now(),
	}
	c.Set(accountTestOpsCollectorContextKey, collector)
	return collector
}

func isAccountTestRuntimeMetaEvent(event TestEvent) bool {
	payload, ok := event.Data.(map[string]any)
	if !ok {
		return false
	}
	kind := strings.TrimSpace(strings.ToLower(strings.TrimSpace(toStringOrEmpty(payload["kind"]))))
	return kind == "runtime_meta"
}

func toStringOrEmpty(value any) string {
	if value == nil {
		return ""
	}
	if v, ok := value.(string); ok {
		return v
	}
	return ""
}

func (s *AccountTestService) captureOpsEvent(c *gin.Context, event TestEvent) {
	collector := s.ensureOpsCollector(c)
	if collector == nil {
		return
	}

	switch strings.TrimSpace(strings.ToLower(event.Type)) {
	case "content":
		if isAccountTestRuntimeMetaEvent(event) {
			return
		}
		text := logredact.RedactText(event.Text, "sso_token")
		if strings.TrimSpace(text) == "" {
			return
		}
		collector.appendPreview(text)
	case "error":
		if strings.TrimSpace(event.Error) != "" {
			collector.ErrorMessage = logredact.RedactText(event.Error, "sso_token")
		}
	}
}

func (c *accountTestOpsCollector) appendPreview(text string) {
	if c == nil {
		return
	}
	if strings.TrimSpace(text) == "" {
		return
	}
	if c.ResponsePreview == "" {
		c.ResponsePreview = text
	} else {
		c.ResponsePreview += text
	}

	if len(c.ResponsePreview) > accountTestOpsResponsePreviewLimitBytes {
		c.ResponsePreview = c.ResponsePreview[:accountTestOpsResponsePreviewLimitBytes]
		c.PreviewTruncated = true
	}
}

func (s *AccountTestService) captureUpstreamFailure(c *gin.Context, statusCode int, body []byte) {
	collector := s.ensureOpsCollector(c)
	if collector == nil {
		return
	}
	collector.UpstreamStatus = &statusCode
	if len(body) == 0 {
		return
	}
	preview := string(body)
	preview = logredact.RedactText(preview, "sso_token")
	preview = strings.TrimSpace(preview)
	if preview == "" {
		return
	}
	if len(preview) > accountTestOpsUpstreamBodyPreviewMaxByte {
		preview = preview[:accountTestOpsUpstreamBodyPreviewMaxByte] + "..."
	}
	collector.UpstreamBodyText = preview
}

func (s *AccountTestService) finalizeUpstreamTrace(
	c *gin.Context,
	accountID int64,
	runtimeMeta accountTestRuntimeMeta,
	requestedModelID string,
	testMode AccountTestMode,
	testErr error,
) {
	if s == nil || s.opsService == nil || c == nil {
		return
	}
	collector := s.ensureOpsCollector(c)
	if collector == nil {
		return
	}

	collector.RuntimeMeta = runtimeMeta
	collector.RequestedModelID = strings.TrimSpace(requestedModelID)
	collector.ResolvedTestMode = string(testMode)

	status := "success"
	statusCode := 200
	if testErr != nil || strings.TrimSpace(collector.ErrorMessage) != "" {
		status = "error"
		statusCode = 500
	}
	if collector.UpstreamStatus != nil {
		statusCode = *collector.UpstreamStatus
	}
	errorText := strings.TrimSpace(collector.ErrorMessage)
	if errorText == "" && testErr != nil {
		errorText = logredact.RedactText(testErr.Error(), "sso_token")
	}

	durationMs := int64(0)
	if !collector.StartedAt.IsZero() {
		durationMs = time.Since(collector.StartedAt).Milliseconds()
	}

	normalizedRequest := map[string]any{
		"test_run_id":        collector.TestRunID,
		"account_id":         accountID,
		"test_mode":          collector.ResolvedTestMode,
		"requested_model_id": collector.RequestedModelID,
		"resolved_model_id":  collector.RuntimeMeta.ResolvedModelID,
		"source_protocol":    collector.RuntimeMeta.SourceProtocol,
		"target_provider":    collector.RuntimeMeta.TargetProvider,
		"target_model_id":    collector.RuntimeMeta.TargetModelID,
		"inbound_endpoint":   collector.RuntimeMeta.InboundEndpoint,
		"compat_path":        collector.RuntimeMeta.CompatPath,
		"simulated_client":   collector.RuntimeMeta.SimulatedClient,
	}
	normalizedRequestBytes, _ := json.Marshal(normalizedRequest)
	normalizedRequestJSON := string(normalizedRequestBytes)

	responsePayload := map[string]any{
		"preview":    collector.ResponsePreview,
		"truncated":  collector.PreviewTruncated,
		"success":    status == "success",
		"error_text": errorText,
	}
	responseBytes, _ := json.Marshal(responsePayload)
	gatewayResponseJSON := string(responseBytes)

	var upstreamResponseJSON *string
	if strings.TrimSpace(collector.UpstreamBodyText) != "" {
		bodyPayload := map[string]any{
			"message":     collector.UpstreamBodyText,
			"http_status": statusCode,
		}
		bodyBytes, _ := json.Marshal(bodyPayload)
		bodyJSON := string(bodyBytes)
		upstreamResponseJSON = &bodyJSON
	}

	trace := GatewayTraceContext{
		Normalize: ProtocolNormalizeResult{
			Platform:       collector.RuntimeMeta.RuntimePlatform,
			ProtocolIn:     collector.RuntimeMeta.SourceProtocol,
			ProtocolOut:    collector.RuntimeMeta.RuntimePlatform,
			Channel:        "admin",
			RoutePath:      firstNonEmptyString(collector.RuntimeMeta.CompatPath, collector.RuntimeMeta.InboundEndpoint),
			UpstreamPath:   collector.RuntimeMeta.InboundEndpoint,
			RequestType:    "probe_action",
			RequestedModel: collector.RequestedModelID,
			UpstreamModel:  firstNonEmptyString(collector.RuntimeMeta.TargetModelID, collector.RuntimeMeta.ResolvedModelID),
			ActualUpstreamModel: firstNonEmptyString(
				collector.RuntimeMeta.TargetModelID,
				collector.RuntimeMeta.ResolvedModelID,
			),
			ProbeAction: collector.ProbeActionBase + "_upstream",
		},
		NormalizedRequestJSON: &normalizedRequestJSON,
		UpstreamResponseJSON:  upstreamResponseJSON,
		GatewayResponseJSON:   &gatewayResponseJSON,
	}

	recordedAt := time.Now().UTC()
	requestID := uuid.NewString()

	var upstreamStatusPtr *int
	if collector.UpstreamStatus != nil {
		v := *collector.UpstreamStatus
		upstreamStatusPtr = &v
	}

	_ = s.opsService.RecordRequestTrace(c.Request.Context(), &OpsRecordRequestTraceInput{
		RequestID:          requestID,
		ClientRequestID:    collector.TestRunID,
		UserID:             collector.AdminUserID,
		AccountID:          &accountID,
		Status:             status,
		StatusCode:         statusCode,
		UpstreamStatusCode: upstreamStatusPtr,
		DurationMs:         durationMs,
		Trace:              trace,
		CreatedAt:          recordedAt,
	})
}

// AttachAccountTestOpsContext wires the test run metadata into gin.Context so that
// AccountTestService can write related ops request traces (action/upstream).
//
// NOTE: This only attaches metadata; it does not enable ops monitoring by itself.
func AttachAccountTestOpsContext(c *gin.Context, testRunID string, probeActionBase string, adminUserID *int64) {
	if c == nil {
		return
	}
	testRunID = strings.TrimSpace(testRunID)
	probeActionBase = strings.TrimSpace(probeActionBase)
	if testRunID != "" {
		c.Set(accountTestOpsTestRunIDContextKey, testRunID)
	}
	if probeActionBase != "" {
		c.Set(accountTestOpsProbeActionBaseContextKey, probeActionBase)
	}
	if adminUserID != nil && *adminUserID > 0 {
		c.Set(accountTestOpsAdminUserIDContextKey, *adminUserID)
	}
}
