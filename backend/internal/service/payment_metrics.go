package service

import "sync/atomic"

type PaymentRuntimeMetricsSnapshot struct {
	CreateSuccess          int64 `json:"create_success"`
	CreateFailure          int64 `json:"create_failure"`
	ProviderLatencyCount   int64 `json:"provider_latency_count"`
	ProviderLatencyMsTotal int64 `json:"provider_latency_ms_total"`
	WebhookSuccess         int64 `json:"webhook_success"`
	WebhookFailure         int64 `json:"webhook_failure"`
	ResumeSuccess          int64 `json:"resume_success"`
	ResumeFailure          int64 `json:"resume_failure"`
	RefundSuccess          int64 `json:"refund_success"`
	RefundFailure          int64 `json:"refund_failure"`
}

var paymentRuntimeMetrics struct {
	createSuccess          atomic.Int64
	createFailure          atomic.Int64
	providerLatencyCount   atomic.Int64
	providerLatencyMsTotal atomic.Int64
	webhookSuccess         atomic.Int64
	webhookFailure         atomic.Int64
	resumeSuccess          atomic.Int64
	resumeFailure          atomic.Int64
	refundSuccess          atomic.Int64
	refundFailure          atomic.Int64
}

func SnapshotPaymentRuntimeMetrics() PaymentRuntimeMetricsSnapshot {
	return PaymentRuntimeMetricsSnapshot{
		CreateSuccess:          paymentRuntimeMetrics.createSuccess.Load(),
		CreateFailure:          paymentRuntimeMetrics.createFailure.Load(),
		ProviderLatencyCount:   paymentRuntimeMetrics.providerLatencyCount.Load(),
		ProviderLatencyMsTotal: paymentRuntimeMetrics.providerLatencyMsTotal.Load(),
		WebhookSuccess:         paymentRuntimeMetrics.webhookSuccess.Load(),
		WebhookFailure:         paymentRuntimeMetrics.webhookFailure.Load(),
		ResumeSuccess:          paymentRuntimeMetrics.resumeSuccess.Load(),
		ResumeFailure:          paymentRuntimeMetrics.resumeFailure.Load(),
		RefundSuccess:          paymentRuntimeMetrics.refundSuccess.Load(),
		RefundFailure:          paymentRuntimeMetrics.refundFailure.Load(),
	}
}

func resetPaymentRuntimeMetricsForTest() {
	paymentRuntimeMetrics.createSuccess.Store(0)
	paymentRuntimeMetrics.createFailure.Store(0)
	paymentRuntimeMetrics.providerLatencyCount.Store(0)
	paymentRuntimeMetrics.providerLatencyMsTotal.Store(0)
	paymentRuntimeMetrics.webhookSuccess.Store(0)
	paymentRuntimeMetrics.webhookFailure.Store(0)
	paymentRuntimeMetrics.resumeSuccess.Store(0)
	paymentRuntimeMetrics.resumeFailure.Store(0)
	paymentRuntimeMetrics.refundSuccess.Store(0)
	paymentRuntimeMetrics.refundFailure.Store(0)
}

func recordPaymentProviderLatency(ms int64) {
	if ms < 0 {
		ms = 0
	}
	paymentRuntimeMetrics.providerLatencyCount.Add(1)
	paymentRuntimeMetrics.providerLatencyMsTotal.Add(ms)
}
