package protocolruntime

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

type MetricsSnapshot struct {
	RouteMismatchTotal                int64            `json:"route_mismatch_total"`
	UnsupportedActionTotal            int64            `json:"unsupported_action_total"`
	LocalizationFallbackTotal         int64            `json:"localization_fallback_total"`
	AccountTestResolutionFailedTotal  int64            `json:"account_test_resolution_failed_total"`
	AccountProbeResolutionFailedTotal int64            `json:"account_probe_resolution_failed_total"`
	RecoveryProbeStartedTotal         int64            `json:"recovery_probe_started_total"`
	RecoveryProbeSuccessTotal         int64            `json:"recovery_probe_success_total"`
	RecoveryProbeRetryTotal           int64            `json:"recovery_probe_retry_total"`
	RecoveryProbeBlacklistedTotal     int64            `json:"recovery_probe_blacklisted_total"`
	BillingResolverTotal              int64            `json:"billing_resolver_total"`
	BillingResolverFallbackTotal      int64            `json:"billing_resolver_fallback_total"`
	BillingDeprecatedAPITotal         int64            `json:"billing_deprecated_api_total"`
	BillingBulkApplyTotal             int64            `json:"billing_bulk_apply_total"`
	GeminiBillingFallbackAppliedTotal int64            `json:"gemini_billing_fallback_applied_total"`
	GeminiBillingFallbackMissTotal    int64            `json:"gemini_billing_fallback_miss_total"`
	RouteMismatchByKind               map[string]int64 `json:"route_mismatch_by_kind"`
	UnsupportedActionByReason         map[string]int64 `json:"unsupported_action_by_reason"`
	LocalizationFallbackByKind        map[string]int64 `json:"localization_fallback_by_kind"`
	AccountTestResolutionByReason     map[string]int64 `json:"account_test_resolution_by_reason"`
	AccountProbeResolutionByReason    map[string]int64 `json:"account_probe_resolution_by_reason"`
	RecoveryProbeStartedByReason      map[string]int64 `json:"recovery_probe_started_by_reason"`
	RecoveryProbeSuccessByReason      map[string]int64 `json:"recovery_probe_success_by_reason"`
	RecoveryProbeRetryByReason        map[string]int64 `json:"recovery_probe_retry_by_reason"`
	RecoveryProbeBlacklistedByReason  map[string]int64 `json:"recovery_probe_blacklisted_by_reason"`
	BillingResolverByPath             map[string]int64 `json:"billing_resolver_by_path"`
	BillingResolverFallbackByReason   map[string]int64 `json:"billing_resolver_fallback_by_reason"`
	BillingDeprecatedAPIByPath        map[string]int64 `json:"billing_deprecated_api_by_path"`
	BillingBulkApplyByOperation       map[string]int64 `json:"billing_bulk_apply_by_operation"`
	GeminiBillingFallbackByReason     map[string]int64 `json:"gemini_billing_fallback_by_reason"`
	GeminiBillingFallbackMissByReason map[string]int64 `json:"gemini_billing_fallback_miss_by_reason"`
}

type metrics struct {
	routeMismatchTotal                atomic.Int64
	unsupportedActionTotal            atomic.Int64
	localizationFallbackTotal         atomic.Int64
	accountTestResolutionFailedTotal  atomic.Int64
	accountProbeResolutionFailedTotal atomic.Int64
	recoveryProbeStartedTotal         atomic.Int64
	recoveryProbeSuccessTotal         atomic.Int64
	recoveryProbeRetryTotal           atomic.Int64
	recoveryProbeBlacklistedTotal     atomic.Int64
	billingResolverTotal              atomic.Int64
	billingResolverFallbackTotal      atomic.Int64
	billingDeprecatedAPITotal         atomic.Int64
	billingBulkApplyTotal             atomic.Int64
	geminiBillingFallbackAppliedTotal atomic.Int64
	geminiBillingFallbackMissTotal    atomic.Int64

	routeMismatchByKind               sync.Map
	unsupportedActionByReason         sync.Map
	localizationFallbackByKind        sync.Map
	accountTestResolutionByReason     sync.Map
	accountProbeResolutionByReason    sync.Map
	recoveryProbeStartedByReason      sync.Map
	recoveryProbeSuccessByReason      sync.Map
	recoveryProbeRetryByReason        sync.Map
	recoveryProbeBlacklistedByReason  sync.Map
	billingResolverByPath             sync.Map
	billingResolverFallbackByReason   sync.Map
	billingDeprecatedAPIByPath        sync.Map
	billingBulkApplyByOperation       sync.Map
	geminiBillingFallbackByReason     sync.Map
	geminiBillingFallbackMissByReason sync.Map
}

var defaultMetrics metrics

func Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		RouteMismatchTotal:                defaultMetrics.routeMismatchTotal.Load(),
		UnsupportedActionTotal:            defaultMetrics.unsupportedActionTotal.Load(),
		LocalizationFallbackTotal:         defaultMetrics.localizationFallbackTotal.Load(),
		AccountTestResolutionFailedTotal:  defaultMetrics.accountTestResolutionFailedTotal.Load(),
		AccountProbeResolutionFailedTotal: defaultMetrics.accountProbeResolutionFailedTotal.Load(),
		RecoveryProbeStartedTotal:         defaultMetrics.recoveryProbeStartedTotal.Load(),
		RecoveryProbeSuccessTotal:         defaultMetrics.recoveryProbeSuccessTotal.Load(),
		RecoveryProbeRetryTotal:           defaultMetrics.recoveryProbeRetryTotal.Load(),
		RecoveryProbeBlacklistedTotal:     defaultMetrics.recoveryProbeBlacklistedTotal.Load(),
		BillingResolverTotal:              defaultMetrics.billingResolverTotal.Load(),
		BillingResolverFallbackTotal:      defaultMetrics.billingResolverFallbackTotal.Load(),
		BillingDeprecatedAPITotal:         defaultMetrics.billingDeprecatedAPITotal.Load(),
		BillingBulkApplyTotal:             defaultMetrics.billingBulkApplyTotal.Load(),
		GeminiBillingFallbackAppliedTotal: defaultMetrics.geminiBillingFallbackAppliedTotal.Load(),
		GeminiBillingFallbackMissTotal:    defaultMetrics.geminiBillingFallbackMissTotal.Load(),
		RouteMismatchByKind:               snapshotCounterMap(&defaultMetrics.routeMismatchByKind),
		UnsupportedActionByReason:         snapshotCounterMap(&defaultMetrics.unsupportedActionByReason),
		LocalizationFallbackByKind:        snapshotCounterMap(&defaultMetrics.localizationFallbackByKind),
		AccountTestResolutionByReason:     snapshotCounterMap(&defaultMetrics.accountTestResolutionByReason),
		AccountProbeResolutionByReason:    snapshotCounterMap(&defaultMetrics.accountProbeResolutionByReason),
		RecoveryProbeStartedByReason:      snapshotCounterMap(&defaultMetrics.recoveryProbeStartedByReason),
		RecoveryProbeSuccessByReason:      snapshotCounterMap(&defaultMetrics.recoveryProbeSuccessByReason),
		RecoveryProbeRetryByReason:        snapshotCounterMap(&defaultMetrics.recoveryProbeRetryByReason),
		RecoveryProbeBlacklistedByReason:  snapshotCounterMap(&defaultMetrics.recoveryProbeBlacklistedByReason),
		BillingResolverByPath:             snapshotCounterMap(&defaultMetrics.billingResolverByPath),
		BillingResolverFallbackByReason:   snapshotCounterMap(&defaultMetrics.billingResolverFallbackByReason),
		BillingDeprecatedAPIByPath:        snapshotCounterMap(&defaultMetrics.billingDeprecatedAPIByPath),
		BillingBulkApplyByOperation:       snapshotCounterMap(&defaultMetrics.billingBulkApplyByOperation),
		GeminiBillingFallbackByReason:     snapshotCounterMap(&defaultMetrics.geminiBillingFallbackByReason),
		GeminiBillingFallbackMissByReason: snapshotCounterMap(&defaultMetrics.geminiBillingFallbackMissByReason),
	}
}

func RecordRouteMismatch(kind string) {
	defaultMetrics.routeMismatchTotal.Add(1)
	incrementCounterMap(&defaultMetrics.routeMismatchByKind, kind)
}

func RecordUnsupportedAction(reason string) {
	defaultMetrics.unsupportedActionTotal.Add(1)
	incrementCounterMap(&defaultMetrics.unsupportedActionByReason, reason)
}

func RecordLocalizationFallback(kind string) {
	defaultMetrics.localizationFallbackTotal.Add(1)
	incrementCounterMap(&defaultMetrics.localizationFallbackByKind, kind)
}

func RecordAccountTestResolutionFailed(reason string) {
	defaultMetrics.accountTestResolutionFailedTotal.Add(1)
	incrementCounterMap(&defaultMetrics.accountTestResolutionByReason, reason)
}

func RecordAccountProbeResolutionFailed(reason string) {
	defaultMetrics.accountProbeResolutionFailedTotal.Add(1)
	incrementCounterMap(&defaultMetrics.accountProbeResolutionByReason, reason)
}

func RecordRecoveryProbeStarted(reason string) {
	defaultMetrics.recoveryProbeStartedTotal.Add(1)
	incrementCounterMap(&defaultMetrics.recoveryProbeStartedByReason, reason)
}

func RecordRecoveryProbeSuccess(reason string) {
	defaultMetrics.recoveryProbeSuccessTotal.Add(1)
	incrementCounterMap(&defaultMetrics.recoveryProbeSuccessByReason, reason)
}

func RecordRecoveryProbeRetry(reason string) {
	defaultMetrics.recoveryProbeRetryTotal.Add(1)
	incrementCounterMap(&defaultMetrics.recoveryProbeRetryByReason, reason)
}

func RecordRecoveryProbeBlacklisted(reason string) {
	defaultMetrics.recoveryProbeBlacklistedTotal.Add(1)
	incrementCounterMap(&defaultMetrics.recoveryProbeBlacklistedByReason, reason)
}

func RecordBillingResolver(path string) {
	defaultMetrics.billingResolverTotal.Add(1)
	incrementCounterMap(&defaultMetrics.billingResolverByPath, path)
}

func RecordBillingResolverFallback(reason string) {
	defaultMetrics.billingResolverFallbackTotal.Add(1)
	incrementCounterMap(&defaultMetrics.billingResolverFallbackByReason, reason)
}

func RecordBillingDeprecatedAPI(path string) {
	defaultMetrics.billingDeprecatedAPITotal.Add(1)
	incrementCounterMap(&defaultMetrics.billingDeprecatedAPIByPath, path)
}

func RecordBillingBulkApply(operation string) {
	defaultMetrics.billingBulkApplyTotal.Add(1)
	incrementCounterMap(&defaultMetrics.billingBulkApplyByOperation, operation)
}

func RecordGeminiBillingFallbackApplied(reason string) {
	defaultMetrics.geminiBillingFallbackAppliedTotal.Add(1)
	incrementCounterMap(&defaultMetrics.geminiBillingFallbackByReason, reason)
}

func RecordGeminiBillingFallbackMiss(reason string) {
	defaultMetrics.geminiBillingFallbackMissTotal.Add(1)
	incrementCounterMap(&defaultMetrics.geminiBillingFallbackMissByReason, reason)
}

func ResetForTest() {
	defaultMetrics.routeMismatchTotal.Store(0)
	defaultMetrics.unsupportedActionTotal.Store(0)
	defaultMetrics.localizationFallbackTotal.Store(0)
	defaultMetrics.accountTestResolutionFailedTotal.Store(0)
	defaultMetrics.accountProbeResolutionFailedTotal.Store(0)
	defaultMetrics.recoveryProbeStartedTotal.Store(0)
	defaultMetrics.recoveryProbeSuccessTotal.Store(0)
	defaultMetrics.recoveryProbeRetryTotal.Store(0)
	defaultMetrics.recoveryProbeBlacklistedTotal.Store(0)
	defaultMetrics.billingResolverTotal.Store(0)
	defaultMetrics.billingResolverFallbackTotal.Store(0)
	defaultMetrics.billingDeprecatedAPITotal.Store(0)
	defaultMetrics.billingBulkApplyTotal.Store(0)
	defaultMetrics.geminiBillingFallbackAppliedTotal.Store(0)
	defaultMetrics.geminiBillingFallbackMissTotal.Store(0)
	resetCounterMap(&defaultMetrics.routeMismatchByKind)
	resetCounterMap(&defaultMetrics.unsupportedActionByReason)
	resetCounterMap(&defaultMetrics.localizationFallbackByKind)
	resetCounterMap(&defaultMetrics.accountTestResolutionByReason)
	resetCounterMap(&defaultMetrics.accountProbeResolutionByReason)
	resetCounterMap(&defaultMetrics.recoveryProbeStartedByReason)
	resetCounterMap(&defaultMetrics.recoveryProbeSuccessByReason)
	resetCounterMap(&defaultMetrics.recoveryProbeRetryByReason)
	resetCounterMap(&defaultMetrics.recoveryProbeBlacklistedByReason)
	resetCounterMap(&defaultMetrics.billingResolverByPath)
	resetCounterMap(&defaultMetrics.billingResolverFallbackByReason)
	resetCounterMap(&defaultMetrics.billingDeprecatedAPIByPath)
	resetCounterMap(&defaultMetrics.billingBulkApplyByOperation)
	resetCounterMap(&defaultMetrics.geminiBillingFallbackByReason)
	resetCounterMap(&defaultMetrics.geminiBillingFallbackMissByReason)
}

func incrementCounterMap(target *sync.Map, key string) {
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		normalized = "unknown"
	}
	counter, _ := target.LoadOrStore(normalized, &atomic.Int64{})
	typedCounter, ok := counter.(*atomic.Int64)
	if !ok {
		return
	}
	incrementAtomicCounter(typedCounter)
}

func snapshotCounterMap(source *sync.Map) map[string]int64 {
	keys := make([]string, 0)
	values := make(map[string]int64)
	source.Range(func(key, value any) bool {
		name, ok := key.(string)
		if !ok {
			return true
		}
		counter, ok := value.(*atomic.Int64)
		if !ok {
			return true
		}
		keys = append(keys, name)
		values[name] = counter.Load()
		return true
	})
	sort.Strings(keys)
	ordered := make(map[string]int64, len(keys))
	for _, key := range keys {
		ordered[key] = values[key]
	}
	return ordered
}

func resetCounterMap(source *sync.Map) {
	source.Range(func(key, _ any) bool {
		source.Delete(key)
		return true
	})
}

func incrementAtomicCounter(counter *atomic.Int64) {
	for {
		current := counter.Load()
		if counter.CompareAndSwap(current, current+1) {
			return
		}
	}
}
