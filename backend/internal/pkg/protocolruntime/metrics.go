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
	RouteMismatchByKind               map[string]int64 `json:"route_mismatch_by_kind"`
	UnsupportedActionByReason         map[string]int64 `json:"unsupported_action_by_reason"`
	LocalizationFallbackByKind        map[string]int64 `json:"localization_fallback_by_kind"`
	AccountTestResolutionByReason     map[string]int64 `json:"account_test_resolution_by_reason"`
	AccountProbeResolutionByReason    map[string]int64 `json:"account_probe_resolution_by_reason"`
}

type metrics struct {
	routeMismatchTotal                atomic.Int64
	unsupportedActionTotal            atomic.Int64
	localizationFallbackTotal         atomic.Int64
	accountTestResolutionFailedTotal  atomic.Int64
	accountProbeResolutionFailedTotal atomic.Int64

	routeMismatchByKind            sync.Map
	unsupportedActionByReason      sync.Map
	localizationFallbackByKind     sync.Map
	accountTestResolutionByReason  sync.Map
	accountProbeResolutionByReason sync.Map
}

var defaultMetrics metrics

func Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		RouteMismatchTotal:                defaultMetrics.routeMismatchTotal.Load(),
		UnsupportedActionTotal:            defaultMetrics.unsupportedActionTotal.Load(),
		LocalizationFallbackTotal:         defaultMetrics.localizationFallbackTotal.Load(),
		AccountTestResolutionFailedTotal:  defaultMetrics.accountTestResolutionFailedTotal.Load(),
		AccountProbeResolutionFailedTotal: defaultMetrics.accountProbeResolutionFailedTotal.Load(),
		RouteMismatchByKind:               snapshotCounterMap(&defaultMetrics.routeMismatchByKind),
		UnsupportedActionByReason:         snapshotCounterMap(&defaultMetrics.unsupportedActionByReason),
		LocalizationFallbackByKind:        snapshotCounterMap(&defaultMetrics.localizationFallbackByKind),
		AccountTestResolutionByReason:     snapshotCounterMap(&defaultMetrics.accountTestResolutionByReason),
		AccountProbeResolutionByReason:    snapshotCounterMap(&defaultMetrics.accountProbeResolutionByReason),
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

func ResetForTest() {
	defaultMetrics.routeMismatchTotal.Store(0)
	defaultMetrics.unsupportedActionTotal.Store(0)
	defaultMetrics.localizationFallbackTotal.Store(0)
	defaultMetrics.accountTestResolutionFailedTotal.Store(0)
	defaultMetrics.accountProbeResolutionFailedTotal.Store(0)
	resetCounterMap(&defaultMetrics.routeMismatchByKind)
	resetCounterMap(&defaultMetrics.unsupportedActionByReason)
	resetCounterMap(&defaultMetrics.localizationFallbackByKind)
	resetCounterMap(&defaultMetrics.accountTestResolutionByReason)
	resetCounterMap(&defaultMetrics.accountProbeResolutionByReason)
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
