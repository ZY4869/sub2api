//go:build unit

package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestNormalizeOpsTraceJSONBPayload_RemovesNullRunesFromParsedJSON(t *testing.T) {
	raw := `{"message":"bad\u0000value","nested":{"text":"ok\u0000"}}`

	result := normalizeOpsTraceJSONBPayload(&raw, "normalized_request", "application/json")

	require.Equal(t, opsTraceJSONBActionSanitized, result.Action)
	require.NotNil(t, result.Value)
	require.JSONEq(t, `{"message":"badvalue","nested":{"text":"ok"}}`, *result.Value)
	require.NotContains(t, *result.Value, `\u0000`)
}

func TestNormalizeOpsTraceJSONBPayload_WrapsInvalidUTF8String(t *testing.T) {
	raw := string([]byte{0xff, 0xfe, 'b', 'a', 'd'})

	result := normalizeOpsTraceJSONBPayload(&raw, "request_headers", "application/json")

	require.Equal(t, opsTraceJSONBActionEnveloped, result.Action)
	require.NotNil(t, result.Value)
	require.True(t, json.Valid([]byte(*result.Value)))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(*result.Value), &envelope))
	require.Equal(t, string(OpsTracePayloadStateCaptured), envelope["state"])
	require.Contains(t, envelope, "payload")
}

func TestNormalizeOpsTraceJSONBPayload_WrapsPlainString(t *testing.T) {
	raw := "plain text payload"

	result := normalizeOpsTraceJSONBPayload(&raw, "tool_trace", "application/json")

	require.Equal(t, opsTraceJSONBActionEnveloped, result.Action)
	require.NotNil(t, result.Value)
	require.True(t, json.Valid([]byte(*result.Value)))

	var envelope map[string]any
	require.NoError(t, json.Unmarshal([]byte(*result.Value), &envelope))
	require.Equal(t, string(OpsTracePayloadStateCaptured), envelope["state"])
	require.Equal(t, raw, envelope["payload"])
}

func TestRecordRequestTraceNormalizesJSONBFieldsBeforeInsert(t *testing.T) {
	var captured *OpsInsertRequestTraceInput
	repo := &opsRepoMock{
		InsertRequestTraceFn: func(ctx context.Context, input *OpsInsertRequestTraceInput) (int64, error) {
			captured = input
			return 1, nil
		},
	}
	svc := &OpsService{
		opsRepo: repo,
		cfg: &config.Config{
			Ops: config.OpsConfig{
				Enabled: true,
				RequestDetails: config.OpsRequestDetailsConfig{
					Enabled: true,
				},
			},
		},
	}

	normalizedRequest := `{"tool":"bad\u0000value"}`
	requestHeaders := `{"x-request-id":"abc\u0000"}`
	toolTrace := "plain text \u0000payload"
	rawRequest := []byte(`{"message":"bad\u0000value"}`)

	err := svc.RecordRequestTrace(context.Background(), &OpsRecordRequestTraceInput{
		RequestID:  "req-jsonb-normalize",
		StatusCode: 200,
		DurationMs: 4001,
		Trace: GatewayTraceContext{
			Normalize: ProtocolNormalizeResult{
				Platform:       PlatformOpenAI,
				ProtocolIn:     PlatformOpenAI,
				ProtocolOut:    PlatformOpenAI,
				RequestType:    "responses",
				RequestedModel: "gpt-5.4",
			},
			NormalizedRequestJSON: &normalizedRequest,
			RequestHeadersJSON:    &requestHeaders,
			ToolTraceJSON:         &toolTrace,
			RawRequest:            rawRequest,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, captured)

	require.NotNil(t, captured.NormalizedRequestJSON)
	require.JSONEq(t, `{"tool":"badvalue"}`, *captured.NormalizedRequestJSON)
	require.NotContains(t, *captured.NormalizedRequestJSON, `\u0000`)

	require.NotNil(t, captured.RequestHeadersJSON)
	require.JSONEq(t, `{"x-request-id":"abc"}`, *captured.RequestHeadersJSON)
	require.NotContains(t, *captured.RequestHeadersJSON, `\u0000`)

	require.NotNil(t, captured.ToolTraceJSON)
	require.True(t, json.Valid([]byte(*captured.ToolTraceJSON)))
	require.NotContains(t, *captured.ToolTraceJSON, `\u0000`)

	require.NotNil(t, captured.InboundRequestJSON)
	require.True(t, json.Valid([]byte(*captured.InboundRequestJSON)))
	require.NotContains(t, *captured.InboundRequestJSON, `\u0000`)
}
