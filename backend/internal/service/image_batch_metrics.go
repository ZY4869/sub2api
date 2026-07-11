package service

import (
	"sync/atomic"
	"time"
)

type ImageBatchRuntimeMetricsSnapshot struct {
	SubmittedJobs             int64   `json:"submitted_jobs"`
	CompletedJobs             int64   `json:"completed_jobs"`
	FailedJobs                int64   `json:"failed_jobs"`
	CancelledJobs             int64   `json:"cancelled_jobs"`
	DownloadFailures          int64   `json:"download_failures"`
	SettlementFailures        int64   `json:"settlement_failures"`
	WorkerRecoveredJobs       int64   `json:"worker_recovered_jobs"`
	OutputsCleanedJobs        int64   `json:"outputs_cleaned_jobs"`
	QueueLagSamples           int64   `json:"queue_lag_samples"`
	QueueLagTotalSeconds      float64 `json:"queue_lag_total_seconds"`
	ExecutionLagSamples       int64   `json:"execution_lag_samples"`
	ExecutionLagTotalSeconds  float64 `json:"execution_lag_total_seconds"`
	SettlementLagSamples      int64   `json:"settlement_lag_samples"`
	SettlementLagTotalSeconds float64 `json:"settlement_lag_total_seconds"`
}

type imageBatchRuntimeMetrics struct {
	submittedJobs            atomic.Int64
	completedJobs            atomic.Int64
	failedJobs               atomic.Int64
	cancelledJobs            atomic.Int64
	downloadFailures         atomic.Int64
	settlementFailures       atomic.Int64
	workerRecoveredJobs      atomic.Int64
	outputsCleanedJobs       atomic.Int64
	queueLagSamples          atomic.Int64
	queueLagMicrosTotal      atomic.Int64
	executionLagSamples      atomic.Int64
	executionLagMicrosTotal  atomic.Int64
	settlementLagSamples     atomic.Int64
	settlementLagMicrosTotal atomic.Int64
}

var defaultImageBatchRuntimeMetrics imageBatchRuntimeMetrics

func ImageBatchRuntimeMetricsSnapshotNow() ImageBatchRuntimeMetricsSnapshot {
	queueSamples := defaultImageBatchRuntimeMetrics.queueLagSamples.Load()
	executionSamples := defaultImageBatchRuntimeMetrics.executionLagSamples.Load()
	settlementSamples := defaultImageBatchRuntimeMetrics.settlementLagSamples.Load()
	return ImageBatchRuntimeMetricsSnapshot{
		SubmittedJobs:             defaultImageBatchRuntimeMetrics.submittedJobs.Load(),
		CompletedJobs:             defaultImageBatchRuntimeMetrics.completedJobs.Load(),
		FailedJobs:                defaultImageBatchRuntimeMetrics.failedJobs.Load(),
		CancelledJobs:             defaultImageBatchRuntimeMetrics.cancelledJobs.Load(),
		DownloadFailures:          defaultImageBatchRuntimeMetrics.downloadFailures.Load(),
		SettlementFailures:        defaultImageBatchRuntimeMetrics.settlementFailures.Load(),
		WorkerRecoveredJobs:       defaultImageBatchRuntimeMetrics.workerRecoveredJobs.Load(),
		OutputsCleanedJobs:        defaultImageBatchRuntimeMetrics.outputsCleanedJobs.Load(),
		QueueLagSamples:           queueSamples,
		QueueLagTotalSeconds:      microsToSeconds(defaultImageBatchRuntimeMetrics.queueLagMicrosTotal.Load()),
		ExecutionLagSamples:       executionSamples,
		ExecutionLagTotalSeconds:  microsToSeconds(defaultImageBatchRuntimeMetrics.executionLagMicrosTotal.Load()),
		SettlementLagSamples:      settlementSamples,
		SettlementLagTotalSeconds: microsToSeconds(defaultImageBatchRuntimeMetrics.settlementLagMicrosTotal.Load()),
	}
}

func recordImageBatchSubmitted() {
	defaultImageBatchRuntimeMetrics.submittedJobs.Add(1)
}

func recordImageBatchCompleted(createdAt time.Time) {
	defaultImageBatchRuntimeMetrics.completedJobs.Add(1)
	recordImageBatchDuration(&defaultImageBatchRuntimeMetrics.executionLagSamples, &defaultImageBatchRuntimeMetrics.executionLagMicrosTotal, createdAt)
}

func recordImageBatchFailed() {
	defaultImageBatchRuntimeMetrics.failedJobs.Add(1)
}

func recordImageBatchCancelled() {
	defaultImageBatchRuntimeMetrics.cancelledJobs.Add(1)
}

func recordImageBatchDownloadFailure() {
	defaultImageBatchRuntimeMetrics.downloadFailures.Add(1)
}

func recordImageBatchSettlementFailure() {
	defaultImageBatchRuntimeMetrics.settlementFailures.Add(1)
}

func recordImageBatchWorkerRecovered() {
	defaultImageBatchRuntimeMetrics.workerRecoveredJobs.Add(1)
}

func recordImageBatchOutputsCleaned(count int64) {
	if count > 0 {
		defaultImageBatchRuntimeMetrics.outputsCleanedJobs.Add(count)
	}
}

func recordImageBatchQueueLag(createdAt time.Time) {
	recordImageBatchDuration(&defaultImageBatchRuntimeMetrics.queueLagSamples, &defaultImageBatchRuntimeMetrics.queueLagMicrosTotal, createdAt)
}

func recordImageBatchSettlementLag(createdAt time.Time) {
	recordImageBatchDuration(&defaultImageBatchRuntimeMetrics.settlementLagSamples, &defaultImageBatchRuntimeMetrics.settlementLagMicrosTotal, createdAt)
}

func recordImageBatchDuration(samples *atomic.Int64, totalMicros *atomic.Int64, startedAt time.Time) {
	if samples == nil || totalMicros == nil || startedAt.IsZero() {
		return
	}
	micros := time.Since(startedAt).Microseconds()
	if micros < 0 {
		return
	}
	samples.Add(1)
	totalMicros.Add(micros)
}
