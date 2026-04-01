package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSnapshotGoogleBatchRuntimeMetrics(t *testing.T) {
	resetGoogleBatchRuntimeMetricsForTest()

	recordGoogleBatchCreateOutcome(true)
	recordGoogleBatchCreateOutcome(false)
	recordGoogleBatchOverflowDecision()
	recordGoogleBatchOverflowDecision()
	recordGoogleBatchOverflowHit()
	recordGoogleBatchArchiveFetchSource("local")
	recordGoogleBatchArchiveFetchSource("official")
	recordGoogleBatchArchiveFetchSource("vertex")
	recordGoogleBatchArchiveFetchSource("unavailable")
	recordGoogleBatchSettlementLag(2 * time.Second)
	recordGoogleBatchSettlementLag(5 * time.Second)
	recordGoogleBatchReservationSaturation(0.5, true)
	recordGoogleBatchReservationSaturation(1.25, false)
	recordGoogleBatchListFanoutLatency(120 * time.Millisecond)
	recordGoogleBatchListFanoutLatency(250 * time.Millisecond)

	snapshot := SnapshotGoogleBatchRuntimeMetrics()
	require.Equal(t, int64(2), snapshot.BatchCreateTotal)
	require.Equal(t, int64(1), snapshot.BatchCreateSuccessTotal)
	require.InDelta(t, 0.5, snapshot.BatchCreateSuccessRate, 0.0001)
	require.Equal(t, int64(2), snapshot.OverflowDecisionTotal)
	require.Equal(t, int64(1), snapshot.OverflowHitTotal)
	require.InDelta(t, 0.5, snapshot.OverflowHitRate, 0.0001)
	require.Equal(t, int64(1), snapshot.ArchiveFetchLocalTotal)
	require.Equal(t, int64(1), snapshot.ArchiveFetchOfficialTotal)
	require.Equal(t, int64(1), snapshot.ArchiveFetchVertexTotal)
	require.Equal(t, int64(1), snapshot.ArchiveFetchUnavailable)
	require.Equal(t, int64(2), snapshot.SettlementLagSamples)
	require.InDelta(t, 7.0, snapshot.SettlementLagTotalSeconds, 0.01)
	require.InDelta(t, 5.0, snapshot.SettlementLagMaxSeconds, 0.01)
	require.Equal(t, int64(2), snapshot.ReservationSaturationSamples)
	require.Equal(t, int64(1), snapshot.ReservationRejectedTotal)
	require.InDelta(t, 0.875, snapshot.ReservationSaturationAvg, 0.001)
	require.InDelta(t, 1.25, snapshot.ReservationSaturationMax, 0.001)
	require.Equal(t, int64(2), snapshot.ListFanoutSamples)
	require.InDelta(t, 185.0, snapshot.ListFanoutAvgMs, 0.01)
	require.InDelta(t, 250.0, snapshot.ListFanoutMaxMs, 0.01)
}
