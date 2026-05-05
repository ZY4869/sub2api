package service

import (
	"sort"
	"strings"
	"sync"
	"sync/atomic"
)

const (
	billingPricingSaveFailureValidation      = "validation"
	billingPricingSaveFailureSnapshotPersist = "snapshot_persist"
	billingPricingSaveFailureOverridePersist = "override_persist"
	billingPricingSaveFailureRulesPersist    = "rules_persist"
	billingPricingSaveFailureCurrencyPersist = "currency_persist"
)

type BillingPricingMetricsSnapshot struct {
	PricingSaveFailedByReason map[string]int64 `json:"pricing_save_failed_by_reason"`
	CNYRuntimeFXBackfillTotal int64            `json:"cny_pricing_runtime_fx_backfill_total"`
}

type billingPricingMetrics struct {
	cnyRuntimeFXBackfillTotal atomic.Int64
	saveFailureCounters       sync.Map
}

var defaultBillingPricingMetrics billingPricingMetrics

func GetBillingPricingMetricsSnapshot() BillingPricingMetricsSnapshot {
	return BillingPricingMetricsSnapshot{
		PricingSaveFailedByReason: snapshotBillingPricingCounterMap(&defaultBillingPricingMetrics.saveFailureCounters),
		CNYRuntimeFXBackfillTotal: defaultBillingPricingMetrics.cnyRuntimeFXBackfillTotal.Load(),
	}
}

func recordBillingPricingSaveFailure(reason string) {
	reason = normalizeBillingPricingSaveFailureReason(reason)
	if reason == "" {
		return
	}
	incrementBillingPricingCounterMap(&defaultBillingPricingMetrics.saveFailureCounters, reason)
}

func recordBillingPricingRuntimeFXBackfillSuccess() {
	defaultBillingPricingMetrics.cnyRuntimeFXBackfillTotal.Add(1)
}

func resetBillingPricingMetricsForTest() {
	defaultBillingPricingMetrics.cnyRuntimeFXBackfillTotal.Store(0)
	defaultBillingPricingMetrics.saveFailureCounters = sync.Map{}
}

func normalizeBillingPricingSaveFailureReason(reason string) string {
	switch strings.TrimSpace(reason) {
	case billingPricingSaveFailureValidation:
		return billingPricingSaveFailureValidation
	case billingPricingSaveFailureSnapshotPersist:
		return billingPricingSaveFailureSnapshotPersist
	case billingPricingSaveFailureOverridePersist:
		return billingPricingSaveFailureOverridePersist
	case billingPricingSaveFailureRulesPersist:
		return billingPricingSaveFailureRulesPersist
	case billingPricingSaveFailureCurrencyPersist:
		return billingPricingSaveFailureCurrencyPersist
	default:
		return ""
	}
}

func incrementBillingPricingCounterMap(counters *sync.Map, key string) {
	if counters == nil || strings.TrimSpace(key) == "" {
		return
	}
	actual, _ := counters.LoadOrStore(key, &atomic.Int64{})
	counter, _ := actual.(*atomic.Int64)
	if counter == nil {
		return
	}
	counter.Add(1)
}

func snapshotBillingPricingCounterMap(counters *sync.Map) map[string]int64 {
	if counters == nil {
		return map[string]int64{}
	}
	result := map[string]int64{}
	keys := make([]string, 0)
	counters.Range(func(key, value any) bool {
		name, ok := key.(string)
		if !ok || strings.TrimSpace(name) == "" {
			return true
		}
		counter, _ := value.(*atomic.Int64)
		if counter == nil {
			return true
		}
		keys = append(keys, name)
		result[name] = counter.Load()
		return true
	})
	sort.Strings(keys)
	ordered := make(map[string]int64, len(keys))
	for _, key := range keys {
		ordered[key] = result[key]
	}
	return ordered
}
