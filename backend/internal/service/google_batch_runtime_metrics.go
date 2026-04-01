package service

import (
	"strings"
	"sync/atomic"
	"time"
)

type GoogleBatchRuntimeMetricsSnapshot struct {
	BatchCreateTotal             int64   `json:"batch_create_total"`
	BatchCreateSuccessTotal      int64   `json:"batch_create_success_total"`
	BatchCreateSuccessRate       float64 `json:"batch_create_success_rate"`
	OverflowDecisionTotal        int64   `json:"overflow_decision_total"`
	OverflowHitTotal             int64   `json:"overflow_hit_total"`
	OverflowHitRate              float64 `json:"overflow_hit_rate"`
	ArchiveFetchLocalTotal       int64   `json:"archive_fetch_local_total"`
	ArchiveFetchOfficialTotal    int64   `json:"archive_fetch_official_total"`
	ArchiveFetchVertexTotal      int64   `json:"archive_fetch_vertex_total"`
	ArchiveFetchUnavailable      int64   `json:"archive_fetch_unavailable_total"`
	SettlementLagSamples         int64   `json:"settlement_lag_samples"`
	SettlementLagTotalSeconds    float64 `json:"settlement_lag_total_seconds"`
	SettlementLagMaxSeconds      float64 `json:"settlement_lag_max_seconds"`
	ReservationSaturationSamples int64   `json:"reservation_saturation_samples"`
	ReservationRejectedTotal     int64   `json:"reservation_rejected_total"`
	ReservationSaturationAvg     float64 `json:"reservation_saturation_avg"`
	ReservationSaturationMax     float64 `json:"reservation_saturation_max"`
	ListFanoutSamples            int64   `json:"list_fanout_samples"`
	ListFanoutAvgMs              float64 `json:"list_fanout_avg_ms"`
	ListFanoutMaxMs              float64 `json:"list_fanout_max_ms"`
}

type googleBatchRuntimeMetrics struct {
	batchCreateTotal          atomic.Int64
	batchCreateSuccessTotal   atomic.Int64
	overflowDecisionTotal     atomic.Int64
	overflowHitTotal          atomic.Int64
	archiveFetchLocalTotal    atomic.Int64
	archiveFetchOfficialTotal atomic.Int64
	archiveFetchVertexTotal   atomic.Int64
	archiveFetchUnavailable   atomic.Int64

	settlementLagSamples     atomic.Int64
	settlementLagMicrosTotal atomic.Int64
	settlementLagMicrosMax   atomic.Int64

	reservationSamples       atomic.Int64
	reservationRejectedTotal atomic.Int64
	reservationMilliTotal    atomic.Int64
	reservationMilliMax      atomic.Int64

	listFanoutSamples     atomic.Int64
	listFanoutMicrosTotal atomic.Int64
	listFanoutMicrosMax   atomic.Int64
}

var defaultGoogleBatchRuntimeMetrics googleBatchRuntimeMetrics

func SnapshotGoogleBatchRuntimeMetrics() GoogleBatchRuntimeMetricsSnapshot {
	createTotal := defaultGoogleBatchRuntimeMetrics.batchCreateTotal.Load()
	createSuccess := defaultGoogleBatchRuntimeMetrics.batchCreateSuccessTotal.Load()
	overflowDecisions := defaultGoogleBatchRuntimeMetrics.overflowDecisionTotal.Load()
	overflowHits := defaultGoogleBatchRuntimeMetrics.overflowHitTotal.Load()
	settlementSamples := defaultGoogleBatchRuntimeMetrics.settlementLagSamples.Load()
	reservationSamples := defaultGoogleBatchRuntimeMetrics.reservationSamples.Load()
	listFanoutSamples := defaultGoogleBatchRuntimeMetrics.listFanoutSamples.Load()

	return GoogleBatchRuntimeMetricsSnapshot{
		BatchCreateTotal:             createTotal,
		BatchCreateSuccessTotal:      createSuccess,
		BatchCreateSuccessRate:       ratioInt64(createSuccess, createTotal),
		OverflowDecisionTotal:        overflowDecisions,
		OverflowHitTotal:             overflowHits,
		OverflowHitRate:              ratioInt64(overflowHits, overflowDecisions),
		ArchiveFetchLocalTotal:       defaultGoogleBatchRuntimeMetrics.archiveFetchLocalTotal.Load(),
		ArchiveFetchOfficialTotal:    defaultGoogleBatchRuntimeMetrics.archiveFetchOfficialTotal.Load(),
		ArchiveFetchVertexTotal:      defaultGoogleBatchRuntimeMetrics.archiveFetchVertexTotal.Load(),
		ArchiveFetchUnavailable:      defaultGoogleBatchRuntimeMetrics.archiveFetchUnavailable.Load(),
		SettlementLagSamples:         settlementSamples,
		SettlementLagTotalSeconds:    microsToSeconds(defaultGoogleBatchRuntimeMetrics.settlementLagMicrosTotal.Load()),
		SettlementLagMaxSeconds:      microsToSeconds(defaultGoogleBatchRuntimeMetrics.settlementLagMicrosMax.Load()),
		ReservationSaturationSamples: reservationSamples,
		ReservationRejectedTotal:     defaultGoogleBatchRuntimeMetrics.reservationRejectedTotal.Load(),
		ReservationSaturationAvg:     ratioInt64(defaultGoogleBatchRuntimeMetrics.reservationMilliTotal.Load(), reservationSamples) / 1000.0,
		ReservationSaturationMax:     float64(defaultGoogleBatchRuntimeMetrics.reservationMilliMax.Load()) / 1000.0,
		ListFanoutSamples:            listFanoutSamples,
		ListFanoutAvgMs:              microsToMillisRatio(defaultGoogleBatchRuntimeMetrics.listFanoutMicrosTotal.Load(), listFanoutSamples),
		ListFanoutMaxMs:              microsToMillis(defaultGoogleBatchRuntimeMetrics.listFanoutMicrosMax.Load()),
	}
}

func recordGoogleBatchCreateOutcome(success bool) {
	defaultGoogleBatchRuntimeMetrics.batchCreateTotal.Add(1)
	if success {
		defaultGoogleBatchRuntimeMetrics.batchCreateSuccessTotal.Add(1)
	}
}

func recordGoogleBatchOverflowDecision() {
	defaultGoogleBatchRuntimeMetrics.overflowDecisionTotal.Add(1)
}

func recordGoogleBatchOverflowHit() {
	defaultGoogleBatchRuntimeMetrics.overflowHitTotal.Add(1)
}

func recordGoogleBatchArchiveFetchSource(source string) {
	switch strings.TrimSpace(source) {
	case "local":
		defaultGoogleBatchRuntimeMetrics.archiveFetchLocalTotal.Add(1)
	case "official":
		defaultGoogleBatchRuntimeMetrics.archiveFetchOfficialTotal.Add(1)
	case "vertex":
		defaultGoogleBatchRuntimeMetrics.archiveFetchVertexTotal.Add(1)
	default:
		defaultGoogleBatchRuntimeMetrics.archiveFetchUnavailable.Add(1)
	}
}

func recordGoogleBatchSettlementLag(duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	micros := duration.Microseconds()
	defaultGoogleBatchRuntimeMetrics.settlementLagSamples.Add(1)
	defaultGoogleBatchRuntimeMetrics.settlementLagMicrosTotal.Add(micros)
	updateAtomicMax(&defaultGoogleBatchRuntimeMetrics.settlementLagMicrosMax, micros)
}

func recordGoogleBatchReservationSaturation(ratio float64, allowed bool) {
	if ratio < 0 {
		ratio = 0
	}
	milli := int64(ratio * 1000)
	defaultGoogleBatchRuntimeMetrics.reservationSamples.Add(1)
	defaultGoogleBatchRuntimeMetrics.reservationMilliTotal.Add(milli)
	updateAtomicMax(&defaultGoogleBatchRuntimeMetrics.reservationMilliMax, milli)
	if !allowed {
		defaultGoogleBatchRuntimeMetrics.reservationRejectedTotal.Add(1)
	}
}

func recordGoogleBatchListFanoutLatency(duration time.Duration) {
	if duration < 0 {
		duration = 0
	}
	micros := duration.Microseconds()
	defaultGoogleBatchRuntimeMetrics.listFanoutSamples.Add(1)
	defaultGoogleBatchRuntimeMetrics.listFanoutMicrosTotal.Add(micros)
	updateAtomicMax(&defaultGoogleBatchRuntimeMetrics.listFanoutMicrosMax, micros)
}

func resetGoogleBatchRuntimeMetricsForTest() {
	defaultGoogleBatchRuntimeMetrics.batchCreateTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.batchCreateSuccessTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.overflowDecisionTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.overflowHitTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.archiveFetchLocalTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.archiveFetchOfficialTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.archiveFetchVertexTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.archiveFetchUnavailable.Store(0)
	defaultGoogleBatchRuntimeMetrics.settlementLagSamples.Store(0)
	defaultGoogleBatchRuntimeMetrics.settlementLagMicrosTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.settlementLagMicrosMax.Store(0)
	defaultGoogleBatchRuntimeMetrics.reservationSamples.Store(0)
	defaultGoogleBatchRuntimeMetrics.reservationRejectedTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.reservationMilliTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.reservationMilliMax.Store(0)
	defaultGoogleBatchRuntimeMetrics.listFanoutSamples.Store(0)
	defaultGoogleBatchRuntimeMetrics.listFanoutMicrosTotal.Store(0)
	defaultGoogleBatchRuntimeMetrics.listFanoutMicrosMax.Store(0)
}

func updateAtomicMax(target *atomic.Int64, candidate int64) {
	for {
		current := target.Load()
		if candidate <= current {
			return
		}
		if target.CompareAndSwap(current, candidate) {
			return
		}
	}
}

func ratioInt64(numerator int64, denominator int64) float64 {
	if denominator <= 0 {
		return 0
	}
	return float64(numerator) / float64(denominator)
}

func microsToSeconds(value int64) float64 {
	return float64(value) / float64(time.Second/time.Microsecond)
}

func microsToMillis(value int64) float64 {
	return float64(value) / 1000.0
}

func microsToMillisRatio(totalMicros int64, samples int64) float64 {
	if samples <= 0 {
		return 0
	}
	return microsToMillis(totalMicros) / float64(samples)
}
