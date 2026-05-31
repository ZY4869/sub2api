package protocolruntime

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestRecordContentModerationDecisionAndAbuseSignalMetrics(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	RecordContentModerationDecision("keyword_block", "keyword_blocked:rule", 7)
	RecordContentModerationDecision("fail_closed", "moderation_timeout", -1)
	RecordAbuseSignal("long_context_request")

	snapshot := Snapshot()
	require.Equal(t, int64(2), snapshot.ContentModerationDecisionTotal)
	require.Equal(t, int64(7), snapshot.ContentModerationDecisionLatencyMsTotal)
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionByResultReason["keyword_block:keyword_blocked:rule"])
	require.Equal(t, int64(7), snapshot.ContentModerationDecisionLatencyMsByResultReason["keyword_block:keyword_blocked:rule"])
	require.Equal(t, int64(1), snapshot.ContentModerationDecisionByResultReason["fail_closed:moderation_timeout"])
	require.Equal(t, int64(1), snapshot.AbuseSignalTotal)
	require.Equal(t, int64(1), snapshot.AbuseSignalByType["long_context_request"])
}

func TestRecordTimePolicyDecisionMetrics(t *testing.T) {
	ResetForTest()
	t.Cleanup(ResetForTest)

	RecordTimePolicyDecision("api_key", true, "allowed", 2)
	RecordTimePolicyDecision("model", false, "outside_window", 3)

	snapshot := Snapshot()
	require.Equal(t, int64(2), snapshot.TimePolicyDecisionTotal)
	require.Equal(t, int64(5), snapshot.TimePolicyDecisionLatencyMsTotal)
	require.Equal(t, int64(1), snapshot.TimePolicyDecisionByScopeResultReason["api_key:allowed:allowed"])
	require.Equal(t, int64(1), snapshot.TimePolicyDecisionByScopeResultReason["model:denied:outside_window"])
	require.Equal(t, int64(3), snapshot.TimePolicyDecisionLatencyMsByScopeResultReason["model:denied:outside_window"])
	require.Equal(t, int64(1), snapshot.TimePolicyDenyTotal)
	require.Equal(t, int64(1), snapshot.TimePolicyDenyByScopeReason["model:outside_window"])
}
