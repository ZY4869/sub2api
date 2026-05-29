package service

import (
	"context"
	"strings"
	"time"
)

func (s *OpsAlertEvaluatorService) computeRuleMetric(
	ctx context.Context,
	rule *OpsAlertRule,
	systemMetrics *OpsSystemMetricsSnapshot,
	start time.Time,
	end time.Time,
	platform string,
	groupID *int64,
	reason *string,
	evalCache *opsAlertEvaluationCache,
) (float64, bool) {
	if rule == nil {
		return 0, false
	}
	metricType := strings.TrimSpace(rule.MetricType)
	if value, ok := computeOpsAlertRuntimeMetric(metricType, reason); ok {
		return value, true
	}
	if value, ok := computeOpsAlertSystemMetric(metricType, systemMetrics); ok {
		return value, true
	}
	if value, ok := s.computeOpsAlertAvailabilityMetric(ctx, metricType, platform, groupID, evalCache); ok {
		return value, true
	}

	overview, err := s.getCachedDashboardOverview(ctx, start, end, platform, groupID, evalCache)
	if err != nil {
		return 0, false
	}
	if overview == nil {
		return 0, false
	}

	switch metricType {
	case "success_rate":
		if overview.RequestCountSLA <= 0 {
			return 0, false
		}
		return overview.SLA * 100, true
	case "error_rate":
		if overview.RequestCountSLA <= 0 {
			return 0, false
		}
		return overview.ErrorRate * 100, true
	case "upstream_error_rate":
		if overview.RequestCountSLA <= 0 {
			return 0, false
		}
		return overview.UpstreamErrorRate * 100, true
	default:
		return 0, false
	}
}

func computeOpsAlertRuntimeMetric(metricType string, reason *string) (float64, bool) {
	switch metricType {
	case "recovery_probe_started_count":
		snapshot := SnapshotProtocolGatewayRuntimeMetrics()
		return selectOpsAlertReasonMetric(snapshot.RecoveryProbeStartedTotal, snapshot.RecoveryProbeStartedByReason, reason), true
	case "recovery_probe_success_count":
		snapshot := SnapshotProtocolGatewayRuntimeMetrics()
		return selectOpsAlertReasonMetric(snapshot.RecoveryProbeSuccessTotal, snapshot.RecoveryProbeSuccessByReason, reason), true
	case "recovery_probe_retry_count":
		snapshot := SnapshotProtocolGatewayRuntimeMetrics()
		return selectOpsAlertReasonMetric(snapshot.RecoveryProbeRetryTotal, snapshot.RecoveryProbeRetryByReason, reason), true
	case "recovery_probe_blacklisted_count":
		snapshot := SnapshotProtocolGatewayRuntimeMetrics()
		return selectOpsAlertReasonMetric(snapshot.RecoveryProbeBlacklistedTotal, snapshot.RecoveryProbeBlacklistedByReason, reason), true
	case "gemini_billing_fallback_applied_count":
		snapshot := SnapshotProtocolGatewayRuntimeMetrics()
		return selectOpsAlertReasonMetric(snapshot.GeminiBillingFallbackAppliedTotal, snapshot.GeminiBillingFallbackByReason, reason), true
	case "gemini_billing_fallback_miss_count":
		snapshot := SnapshotProtocolGatewayRuntimeMetrics()
		return selectOpsAlertReasonMetric(snapshot.GeminiBillingFallbackMissTotal, snapshot.GeminiBillingFallbackMissByReason, reason), true
	default:
		return 0, false
	}
}

func computeOpsAlertSystemMetric(metricType string, systemMetrics *OpsSystemMetricsSnapshot) (float64, bool) {
	switch metricType {
	case "cpu_usage_percent":
		if systemMetrics != nil && systemMetrics.CPUUsagePercent != nil {
			return *systemMetrics.CPUUsagePercent, true
		}
	case "memory_usage_percent":
		if systemMetrics != nil && systemMetrics.MemoryUsagePercent != nil {
			return *systemMetrics.MemoryUsagePercent, true
		}
	case "concurrency_queue_depth":
		if systemMetrics != nil && systemMetrics.ConcurrencyQueueDepth != nil {
			return float64(*systemMetrics.ConcurrencyQueueDepth), true
		}
	}
	return 0, false
}

func selectOpsAlertReasonMetric(total int64, buckets map[string]int64, reason *string) float64 {
	if reason == nil || strings.TrimSpace(*reason) == "" {
		return float64(total)
	}
	if buckets == nil {
		return 0
	}
	return float64(buckets[strings.TrimSpace(*reason)])
}

func compareMetric(value float64, operator string, threshold float64) bool {
	switch strings.TrimSpace(operator) {
	case ">":
		return value > threshold
	case ">=":
		return value >= threshold
	case "<":
		return value < threshold
	case "<=":
		return value <= threshold
	case "==":
		return value == threshold
	case "!=":
		return value != threshold
	default:
		return false
	}
}

func computeGroupAvailableRatio(group *GroupAvailability) float64 {
	if group == nil || group.TotalAccounts <= 0 {
		return 0
	}
	return (float64(group.AvailableCount) / float64(group.TotalAccounts)) * 100
}

func countAccountsByCondition(accounts map[int64]*AccountAvailability, condition func(*AccountAvailability) bool) int64 {
	if len(accounts) == 0 || condition == nil {
		return 0
	}
	var count int64
	for _, account := range accounts {
		if account != nil && condition(account) {
			count++
		}
	}
	return count
}
