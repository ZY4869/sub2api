package admin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func newOpsProtocolGatewayRuntimeRouter(handler *OpsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/runtime/protocol-gateway", handler.GetProtocolGatewayRuntimeMetrics)
	return r
}

func TestOpsProtocolGatewayRuntimeHandler_GetSnapshot(t *testing.T) {
	protocolruntime.ResetForTest()
	protocolruntime.RecordRouteMismatch("unknown_public_endpoint")
	protocolruntime.RecordUnsupportedAction(service.GatewayReasonUnsupportedAction)
	protocolruntime.RecordLocalizationFallback("message_key:en")
	protocolruntime.RecordAccountTestResolutionFailed("TEST_TARGET_MODEL_INVALID")
	protocolruntime.RecordAccountProbeResolutionFailed("TEST_PROBE_RESOLUTION_FAILED")
	t.Cleanup(protocolruntime.ResetForTest)

	h := NewOpsHandler(newRuntimeOpsService(t))
	r := newOpsProtocolGatewayRuntimeRouter(h)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/runtime/protocol-gateway", nil)
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d, want 200", w.Code)
	}

	var resp response.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Code != 0 {
		t.Fatalf("code=%d, want 0", resp.Code)
	}

	raw, err := json.Marshal(resp.Data)
	if err != nil {
		t.Fatalf("marshal response data: %v", err)
	}

	var snapshot service.ProtocolGatewayRuntimeMetricsSnapshot
	if err := json.Unmarshal(raw, &snapshot); err != nil {
		t.Fatalf("unmarshal snapshot: %v", err)
	}
	if snapshot.RouteMismatchTotal != 1 {
		t.Fatalf("route_mismatch_total=%d, want 1", snapshot.RouteMismatchTotal)
	}
	if snapshot.UnsupportedActionTotal != 1 {
		t.Fatalf("unsupported_action_total=%d, want 1", snapshot.UnsupportedActionTotal)
	}
	if snapshot.LocalizationFallbackTotal != 1 {
		t.Fatalf("localization_fallback_total=%d, want 1", snapshot.LocalizationFallbackTotal)
	}
	if snapshot.AccountTestResolutionFailedTotal != 1 {
		t.Fatalf("account_test_resolution_failed_total=%d, want 1", snapshot.AccountTestResolutionFailedTotal)
	}
	if snapshot.AccountProbeResolutionFailedTotal != 1 {
		t.Fatalf("account_probe_resolution_failed_total=%d, want 1", snapshot.AccountProbeResolutionFailedTotal)
	}
	if snapshot.RouteMismatchByKind["unknown_public_endpoint"] != 1 {
		t.Fatalf("route_mismatch_by_kind=%v", snapshot.RouteMismatchByKind)
	}
	if snapshot.UnsupportedActionByReason[service.GatewayReasonUnsupportedAction] != 1 {
		t.Fatalf("unsupported_action_by_reason=%v", snapshot.UnsupportedActionByReason)
	}
	if snapshot.LocalizationFallbackByKind["message_key:en"] != 1 {
		t.Fatalf("localization_fallback_by_kind=%v", snapshot.LocalizationFallbackByKind)
	}
	if snapshot.AccountTestResolutionByReason["TEST_TARGET_MODEL_INVALID"] != 1 {
		t.Fatalf("account_test_resolution_by_reason=%v", snapshot.AccountTestResolutionByReason)
	}
	if snapshot.AccountProbeResolutionByReason["TEST_PROBE_RESOLUTION_FAILED"] != 1 {
		t.Fatalf("account_probe_resolution_by_reason=%v", snapshot.AccountProbeResolutionByReason)
	}
}
