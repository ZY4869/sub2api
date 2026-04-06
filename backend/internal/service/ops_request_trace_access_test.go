//go:build unit

package service

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestGetRequestTraceByIDMarksAdminRawAccessAllowed(t *testing.T) {
	repo := &opsRepoMock{
		GetRequestTraceByIDFn: func(ctx context.Context, id int64) (*OpsRequestTraceDetail, error) {
			return &OpsRequestTraceDetail{
				OpsRequestTraceListItem: OpsRequestTraceListItem{
					ID:           id,
					RequestID:    "req-admin",
					RawAvailable: true,
				},
			}, nil
		},
		ListRequestTraceAuditsFn: func(ctx context.Context, traceID int64) ([]*OpsRequestTraceAuditLog, error) {
			return []*OpsRequestTraceAuditLog{}, nil
		},
	}
	svc := &OpsService{
		opsRepo: repo,
		cfg: &config.Config{
			Ops: config.OpsConfig{
				Enabled: true,
				RequestDetails: config.OpsRequestDetailsConfig{
					Enabled:       true,
					EncryptionKey: "trace-secret",
				},
			},
		},
	}

	detail, err := svc.GetRequestTraceByID(WithOpsRequestTraceAdminRawAccess(context.Background(), true), 101, 7)
	require.NoError(t, err)
	require.True(t, detail.RawAccessAllowed)

	detail, err = svc.GetRequestTraceByID(context.Background(), 101, 7)
	require.NoError(t, err)
	require.False(t, detail.RawAccessAllowed)
}

func TestGetRequestTraceRawByIDAllowsAdminContext(t *testing.T) {
	rawRequest := []byte(`{"messages":[{"role":"user","content":"hello"}]}`)
	rawCiphertext, _, _, err := buildEncryptedTracePayload("trace-secret", rawRequest, 512*1024)
	require.NoError(t, err)

	repo := &opsRepoMock{
		GetRequestTraceRawByIDFn: func(ctx context.Context, id int64) (*OpsRequestTraceRawDetail, error) {
			return &OpsRequestTraceRawDetail{
				ID:          id,
				RequestID:   "req-admin-raw",
				RawRequest:  string(rawCiphertext),
				RawResponse: "",
			}, nil
		},
		InsertRequestTraceAuditFn: func(ctx context.Context, input *OpsInsertRequestTraceAuditInput) error {
			return nil
		},
	}
	svc := &OpsService{
		opsRepo: repo,
		cfg: &config.Config{
			Ops: config.OpsConfig{
				Enabled: true,
				RequestDetails: config.OpsRequestDetailsConfig{
					Enabled:       true,
					EncryptionKey: "trace-secret",
				},
			},
		},
	}

	_, err = svc.GetRequestTraceRawByID(context.Background(), 101, 7)
	require.Error(t, err)

	detail, err := svc.GetRequestTraceRawByID(WithOpsRequestTraceAdminRawAccess(context.Background(), true), 101, 7)
	require.NoError(t, err)
	require.JSONEq(t, string(rawRequest), detail.RawRequest)
}

func TestRecordRequestTraceKeepsExpandedInboundPreview(t *testing.T) {
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

	largePrompt := strings.Repeat("hello ", 50000)
	rawRequest, err := json.Marshal(map[string]any{
		"model":  "gpt-5.4",
		"stream": true,
		"input":  largePrompt,
	})
	require.NoError(t, err)

	err = svc.RecordRequestTrace(context.Background(), &OpsRecordRequestTraceInput{
		RequestID:  "req-preview",
		StatusCode: 200,
		DurationMs: 12,
		Trace: GatewayTraceContext{
			Normalize: ProtocolNormalizeResult{
				Platform:       PlatformOpenAI,
				ProtocolIn:     PlatformOpenAI,
				ProtocolOut:    PlatformOpenAI,
				RequestType:    "responses",
				RequestedModel: "gpt-5.4",
				Stream:         true,
			},
			RawRequest: rawRequest,
		},
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.NotNil(t, captured.InboundRequestJSON)
	require.Contains(t, *captured.InboundRequestJSON, "\"input\"")
	require.NotContains(t, *captured.InboundRequestJSON, "\"request_body_truncated\":true")
}

func TestRecordRequestTraceNormalizesEmptyToolKinds(t *testing.T) {
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

	err := svc.RecordRequestTrace(context.Background(), &OpsRecordRequestTraceInput{
		RequestID:  "req-no-tools",
		StatusCode: 200,
		DurationMs: 3500,
		Trace: GatewayTraceContext{
			Normalize: ProtocolNormalizeResult{
				Platform:       PlatformOpenAI,
				ProtocolIn:     PlatformOpenAI,
				ProtocolOut:    PlatformOpenAI,
				RequestType:    "responses",
				RequestedModel: "gpt-5.4",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.NotNil(t, captured.ToolKinds)
	require.Empty(t, captured.ToolKinds)
}
