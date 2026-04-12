package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/protocolruntime"
	"github.com/stretchr/testify/require"
)

func TestRecordGeminiBillingRuntimeMetrics_FallbackApplied(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	recordGeminiBillingRuntimeMetrics(&BillingSimulationResult{}, &BillingSimulationFallback{
		Applied: true,
		Reason:  "no_billing_rule_match",
	})

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.GeminiBillingFallbackAppliedTotal)
	require.Equal(t, int64(0), snapshot.GeminiBillingFallbackMissTotal)
	require.Equal(t, int64(1), snapshot.GeminiBillingFallbackByReason["no_billing_rule_match"])
}

func TestRecordGeminiBillingRuntimeMetrics_UsesUnmatchedReasonsWhenFallbackNotApplied(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	recordGeminiBillingRuntimeMetrics(&BillingSimulationResult{
		UnmatchedDemands: []BillingSimulationUnmatchedDemand{
			{Reason: "surface_miss"},
			{Reason: "surface_miss"},
			{Reason: "model_matcher_miss"},
		},
	}, nil)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(0), snapshot.GeminiBillingFallbackAppliedTotal)
	require.Equal(t, int64(2), snapshot.GeminiBillingFallbackMissTotal)
	require.Equal(t, int64(1), snapshot.GeminiBillingFallbackMissByReason["surface_miss"])
	require.Equal(t, int64(1), snapshot.GeminiBillingFallbackMissByReason["model_matcher_miss"])
}

func TestRecordGeminiBillingRuntimeMetrics_TracksMatchedRulesAsFallbackMissReason(t *testing.T) {
	protocolruntime.ResetForTest()
	t.Cleanup(protocolruntime.ResetForTest)

	recordGeminiBillingRuntimeMetrics(&BillingSimulationResult{
		Lines: []BillingSimulationLine{{ChargeSlot: BillingChargeSlotTextInput, Unit: BillingUnitInputToken, Units: 128}},
	}, nil)

	snapshot := protocolruntime.Snapshot()
	require.Equal(t, int64(1), snapshot.GeminiBillingFallbackMissTotal)
	require.Equal(t, int64(1), snapshot.GeminiBillingFallbackMissByReason["rules_matched"])
}
