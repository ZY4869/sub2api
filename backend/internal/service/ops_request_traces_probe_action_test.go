//go:build unit

package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/stretchr/testify/require"
)

func TestRecordRequestTrace_ForcesCaptureWhenProbeActionIsSet(t *testing.T) {
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
					Enabled:           true,
					SuccessSampleRate: 0,
					ForceCaptureSlowMs: 999999,
				},
			},
		},
	}

	err := svc.RecordRequestTrace(context.Background(), &OpsRecordRequestTraceInput{
		RequestID:  "req-probe-action",
		StatusCode: 200,
		DurationMs: 1,
		Trace: GatewayTraceContext{
			Normalize: ProtocolNormalizeResult{
				Platform:       PlatformOpenAI,
				ProtocolIn:     PlatformOpenAI,
				ProtocolOut:    PlatformOpenAI,
				RequestType:    "responses",
				RequestedModel: "gpt-5.4",
				ProbeAction:    "account_test",
			},
		},
	})
	require.NoError(t, err)
	require.NotNil(t, captured)
	require.Equal(t, "probe_action", captured.CaptureReason)
	require.Equal(t, "account_test", captured.ProbeAction)
}

