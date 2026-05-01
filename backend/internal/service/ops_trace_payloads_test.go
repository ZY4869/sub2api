package service

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSetOpsTraceGatewayResponseCompactsLargePayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	body := []byte(`{"created":123,"data":[{"b64_json":"` + strings.Repeat("A", opsTracePayloadInlineBytesLimit+1) + `"}]}`)

	SetOpsTraceGatewayResponse(c, "image_response", body, "application/json", false)

	got := GetOpsTraceGatewayResponseJSON(c)
	if got == nil {
		t.Fatalf("expected compacted trace payload")
	}
	if strings.Contains(*got, strings.Repeat("A", 128)) {
		t.Fatalf("large base64 payload was not omitted")
	}
	if !strings.Contains(*got, `"omitted":true`) {
		t.Fatalf("compacted payload missing omitted marker: %s", *got)
	}
	if !strings.Contains(*got, `"payload_exceeds_preview_limit"`) {
		t.Fatalf("compacted payload missing reason: %s", *got)
	}
}

func TestBuildOpsTracePayloadEnvelopeJSONCompactsLargeStrings(t *testing.T) {
	payload := map[string]any{
		"output_text": strings.Repeat("x", opsTraceLargeStringBytesLimit+1),
	}

	got := BuildOpsTracePayloadEnvelopeJSON(OpsTracePayloadStateCaptured, "large_text", payload, "application/json", false)

	if got == nil {
		t.Fatalf("expected trace payload")
	}
	if strings.Contains(*got, strings.Repeat("x", 128)) {
		t.Fatalf("large string payload was not omitted")
	}
	if !strings.Contains(*got, `"large_string_omitted"`) {
		t.Fatalf("compacted payload missing large string marker: %s", *got)
	}
}

func TestSetOpsTraceGatewayResponseUsesConfiguredPreviewLimit(t *testing.T) {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(httptest.NewRecorder())
	SetOpsTracePayloadPreviewLimit(c, 32)

	SetOpsTraceGatewayResponse(
		c,
		"preview_limited_response",
		[]byte(`{"model":"deepseek-v4-pro","reasoning_effort":"high","payload":"`+strings.Repeat("A", 128)+`"}`),
		"application/json",
		false,
	)

	got := GetOpsTraceGatewayResponseJSON(c)
	if got == nil {
		t.Fatalf("expected trace payload")
	}
	if !strings.Contains(*got, `"preview_limit_bytes":32`) {
		t.Fatalf("expected configured preview limit to be applied: %s", *got)
	}
	if !strings.Contains(*got, `"payload_exceeds_preview_limit"`) {
		t.Fatalf("expected payload omission marker: %s", *got)
	}
}
