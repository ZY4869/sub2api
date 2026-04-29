package service

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

type modelDebugHTTPClientFunc func(req *http.Request) (*http.Response, error)

func (f modelDebugHTTPClientFunc) Do(req *http.Request) (*http.Response, error) {
	return f(req)
}

type modelDebugAPIKeyReaderTestStub struct {
	apiKey *APIKey
}

func (s modelDebugAPIKeyReaderTestStub) GetByID(context.Context, int64) (*APIKey, error) {
	return s.apiKey, nil
}

func TestModelDebugService_Run_RejectsInvalidEndpointCombo(t *testing.T) {
	svc := NewModelDebugService(modelDebugAPIKeyReaderTestStub{}, nil, &config.Config{})
	events := make([]string, 0, 2)

	err := svc.Run(context.Background(), ModelDebugRunInput{
		AdminUserID:  1,
		BaseURL:      "http://example.test",
		KeyMode:      ModelDebugKeyModeManual,
		ManualAPIKey: "manual-secret",
		Protocol:     ModelDebugProtocolAnthropic,
		EndpointKind: ModelDebugEndpointResponses,
		Model:        "claude-sonnet-4.5",
		RequestBody:  map[string]any{"messages": []any{}},
	}, func(event string, payload any) error {
		events = append(events, event)
		return nil
	})

	require.Error(t, err)
	require.Equal(t, []string{modelDebugEventStart, modelDebugEventError}, events)
}

func TestModelDebugService_Run_StopsOnCanceledRequest(t *testing.T) {
	svc := NewModelDebugService(modelDebugAPIKeyReaderTestStub{}, nil, &config.Config{})
	svc.SetHTTPClient(modelDebugHTTPClientFunc(func(req *http.Request) (*http.Response, error) {
		<-req.Context().Done()
		return nil, req.Context().Err()
	}))

	ctx, cancel := context.WithCancel(context.Background())
	events := make([]string, 0, 4)
	cancel()

	err := svc.Run(ctx, ModelDebugRunInput{
		AdminUserID:  1,
		BaseURL:      "http://example.test",
		KeyMode:      ModelDebugKeyModeManual,
		ManualAPIKey: "manual-secret",
		Protocol:     ModelDebugProtocolOpenAI,
		EndpointKind: ModelDebugEndpointResponses,
		Model:        "gpt-5.4",
		RequestBody:  map[string]any{"input": "hello"},
	}, func(event string, payload any) error {
		events = append(events, event)
		return nil
	})

	require.Error(t, err)
	require.True(t, errors.Is(err, context.Canceled))
	require.Equal(t, []string{modelDebugEventStart, modelDebugEventRequest, modelDebugEventError}, events)
}

func TestModelDebugService_Run_RecordsDebugTraceMetadata(t *testing.T) {
	var captured []*OpsInsertRequestTraceInput
	opsSvc := &OpsService{
		opsRepo: &opsRepoMock{
			InsertRequestTraceFn: func(ctx context.Context, input *OpsInsertRequestTraceInput) (int64, error) {
				cloned := *input
				captured = append(captured, &cloned)
				return int64(len(captured)), nil
			},
		},
		cfg: &config.Config{
			Ops: config.OpsConfig{
				Enabled: true,
				RequestDetails: config.OpsRequestDetailsConfig{
					Enabled:            true,
					SuccessSampleRate:  0,
					ForceCaptureSlowMs: 999999,
				},
			},
		},
	}

	svc := NewModelDebugService(modelDebugAPIKeyReaderTestStub{}, opsSvc, &config.Config{})
	svc.SetHTTPClient(modelDebugHTTPClientFunc(func(req *http.Request) (*http.Response, error) {
		require.Equal(t, "Bearer manual-secret", req.Header.Get("Authorization"))
		resp := &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(strings.NewReader("{\"id\":\"resp_1\",\"output_text\":\"ok\"}")),
		}
		resp.Header.Set("X-Request-Id", "upstream-debug-1")
		return resp, nil
	}))

	var debugRunID string
	err := svc.Run(context.Background(), ModelDebugRunInput{
		AdminUserID:     42,
		BaseURL:         "http://example.test",
		ClientRequestID: "client-debug-1",
		KeyMode:         ModelDebugKeyModeManual,
		ManualAPIKey:    "manual-secret",
		Protocol:        ModelDebugProtocolOpenAI,
		EndpointKind:    ModelDebugEndpointResponses,
		Model:           "gpt-5.4",
		Stream:          false,
		RequestBody:     map[string]any{"input": "hello"},
	}, func(event string, payload any) error {
		if event == modelDebugEventStart {
			startPayload, ok := payload.(map[string]any)
			require.True(t, ok)
			debugRunID, _ = startPayload["debug_run_id"].(string)
		}
		return nil
	})

	require.NoError(t, err)
	require.NotEmpty(t, debugRunID)
	require.Len(t, captured, 2)
	require.Equal(t, modelDebugTraceAction, captured[0].ProbeAction)
	require.Equal(t, modelDebugUpstreamAction, captured[1].ProbeAction)
	for _, trace := range captured {
		require.Equal(t, debugRunID, trace.RequestID)
		require.Equal(t, "client-debug-1", trace.ClientRequestID)
		require.Equal(t, "upstream-debug-1", trace.UpstreamRequestID)
		require.Equal(t, "/api/v1/admin/models/debug/run", trace.RoutePath)
		require.Equal(t, "/v1/responses", trace.UpstreamPath)
		require.Equal(t, "model_debug", trace.RequestType)
		require.Equal(t, "gpt-5.4", trace.RequestedModel)
		require.NotNil(t, trace.NormalizedRequestJSON)
		require.Contains(t, *trace.NormalizedRequestJSON, "[REDACTED]")
		require.NotContains(t, *trace.NormalizedRequestJSON, "manual-secret")
	}
}
