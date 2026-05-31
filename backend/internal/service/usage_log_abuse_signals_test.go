package service

import (
	"sync"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

func TestRecordUsageLogAbuseSignals_LongContextAndHighRiskSwitch(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)
	usageLogLastHighRiskModelBySubject = syncMapForUsageLogAbuseSignalTest()
	t.Cleanup(func() {
		usageLogLastHighRiskModelBySubject = syncMapForUsageLogAbuseSignalTest()
	})

	longContext := 200_000
	RecordUsageLogAbuseSignals(&UsageLog{
		UserID:                     7,
		APIKeyID:                   11,
		RequestID:                  "req-long",
		RequestedModel:             "gpt-5.4",
		RequestContextLengthTokens: &longContext,
	})
	RecordUsageLogAbuseSignals(&UsageLog{
		UserID:         7,
		APIKeyID:       11,
		RequestID:      "req-switch",
		RequestedModel: "claude-opus-4-8",
	})

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(2), snapshot.AbuseSignalTotal)
	require.Equal(t, int64(1), snapshot.AbuseSignalByType["long_context_request"])
	require.Equal(t, int64(1), snapshot.AbuseSignalByType["high_risk_model_switch"])
}

func syncMapForUsageLogAbuseSignalTest() sync.Map {
	return sync.Map{}
}
