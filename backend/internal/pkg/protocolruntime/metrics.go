package protocolruntime

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
)

type MetricsSnapshot struct {
	RouteMismatchTotal                     int64            `json:"route_mismatch_total"`
	UnsupportedActionTotal                 int64            `json:"unsupported_action_total"`
	LocalizationFallbackTotal              int64            `json:"localization_fallback_total"`
	AccountTestResolutionFailedTotal       int64            `json:"account_test_resolution_failed_total"`
	AccountProbeResolutionFailedTotal      int64            `json:"account_probe_resolution_failed_total"`
	RecoveryProbeStartedTotal              int64            `json:"recovery_probe_started_total"`
	RecoveryProbeSuccessTotal              int64            `json:"recovery_probe_success_total"`
	RecoveryProbeRetryTotal                int64            `json:"recovery_probe_retry_total"`
	RecoveryProbeBlacklistedTotal          int64            `json:"recovery_probe_blacklisted_total"`
	BillingResolverTotal                   int64            `json:"billing_resolver_total"`
	BillingResolverFallbackTotal           int64            `json:"billing_resolver_fallback_total"`
	BillingDeprecatedAPITotal              int64            `json:"billing_deprecated_api_total"`
	BillingBulkApplyTotal                  int64            `json:"billing_bulk_apply_total"`
	GeminiBillingFallbackAppliedTotal      int64            `json:"gemini_billing_fallback_applied_total"`
	GeminiBillingFallbackMissTotal         int64            `json:"gemini_billing_fallback_miss_total"`
	PublicModelProjectionTotal             int64            `json:"public_model_projection_total"`
	PublicModelRestrictionHitTotal         int64            `json:"public_model_restriction_hit_total"`
	ImageRouteTotal                        int64            `json:"image_route_total"`
	ImageRouteSuccessTotal                 int64            `json:"image_route_success_total"`
	ImageRouteFailureTotal                 int64            `json:"image_route_failure_total"`
	ImageRouteLatencyMsTotal               int64            `json:"image_route_latency_ms_total"`
	ResponsesImagegenCompatTotal           int64            `json:"responses_imagegen_compat_total"`
	ResponsesImagegenNormalizedTotal       int64            `json:"responses_imagegen_normalized_total"`
	ResponsesImagegenRejectTotal           int64            `json:"responses_imagegen_reject_total"`
	ResponsesImageToolFailureTotal         int64            `json:"responses_image_tool_failure_total"`
	RouteMismatchByKind                    map[string]int64 `json:"route_mismatch_by_kind"`
	UnsupportedActionByReason              map[string]int64 `json:"unsupported_action_by_reason"`
	LocalizationFallbackByKind             map[string]int64 `json:"localization_fallback_by_kind"`
	AccountTestResolutionByReason          map[string]int64 `json:"account_test_resolution_by_reason"`
	AccountProbeResolutionByReason         map[string]int64 `json:"account_probe_resolution_by_reason"`
	RecoveryProbeStartedByReason           map[string]int64 `json:"recovery_probe_started_by_reason"`
	RecoveryProbeSuccessByReason           map[string]int64 `json:"recovery_probe_success_by_reason"`
	RecoveryProbeRetryByReason             map[string]int64 `json:"recovery_probe_retry_by_reason"`
	RecoveryProbeBlacklistedByReason       map[string]int64 `json:"recovery_probe_blacklisted_by_reason"`
	BillingResolverByPath                  map[string]int64 `json:"billing_resolver_by_path"`
	BillingResolverFallbackByReason        map[string]int64 `json:"billing_resolver_fallback_by_reason"`
	BillingDeprecatedAPIByPath             map[string]int64 `json:"billing_deprecated_api_by_path"`
	BillingBulkApplyByOperation            map[string]int64 `json:"billing_bulk_apply_by_operation"`
	GeminiBillingFallbackByReason          map[string]int64 `json:"gemini_billing_fallback_by_reason"`
	GeminiBillingFallbackMissByReason      map[string]int64 `json:"gemini_billing_fallback_miss_by_reason"`
	PublicModelProjectionBySource          map[string]int64 `json:"public_model_projection_by_source"`
	PublicModelRestrictionByReason         map[string]int64 `json:"public_model_restriction_by_reason"`
	ImageRouteByFamily                     map[string]int64 `json:"image_route_by_family"`
	ImageRouteByProvider                   map[string]int64 `json:"image_route_by_provider"`
	ImageRouteSuccessByFamily              map[string]int64 `json:"image_route_success_by_family"`
	ImageRouteSuccessByProvider            map[string]int64 `json:"image_route_success_by_provider"`
	ImageRouteFailureByFamily              map[string]int64 `json:"image_route_failure_by_family"`
	ImageRouteFailureByProvider            map[string]int64 `json:"image_route_failure_by_provider"`
	ImageRouteLatencyMsByFamily            map[string]int64 `json:"image_route_latency_ms_by_family"`
	ImageRouteLatencyMsByProvider          map[string]int64 `json:"image_route_latency_ms_by_provider"`
	ImageRouteByProtocolMode               map[string]int64 `json:"image_route_by_protocol_mode"`
	ImageRouteSuccessByProtocolMode        map[string]int64 `json:"image_route_success_by_protocol_mode"`
	ImageRouteFailureByProtocolMode        map[string]int64 `json:"image_route_failure_by_protocol_mode"`
	ImageRouteLatencyMsByProtocolMode      map[string]int64 `json:"image_route_latency_ms_by_protocol_mode"`
	ImageRouteByAction                     map[string]int64 `json:"image_route_by_action"`
	ImageRouteSuccessByAction              map[string]int64 `json:"image_route_success_by_action"`
	ImageRouteFailureByAction              map[string]int64 `json:"image_route_failure_by_action"`
	ImageRouteLatencyMsByAction            map[string]int64 `json:"image_route_latency_ms_by_action"`
	ImageRouteBySizeTier                   map[string]int64 `json:"image_route_by_size_tier"`
	ImageRouteSuccessBySizeTier            map[string]int64 `json:"image_route_success_by_size_tier"`
	ImageRouteFailureBySizeTier            map[string]int64 `json:"image_route_failure_by_size_tier"`
	ImageRouteLatencyMsBySizeTier          map[string]int64 `json:"image_route_latency_ms_by_size_tier"`
	ImageRouteByCapabilityProfile          map[string]int64 `json:"image_route_by_capability_profile"`
	ImageRouteSuccessByCapabilityProfile   map[string]int64 `json:"image_route_success_by_capability_profile"`
	ImageRouteFailureByCapabilityProfile   map[string]int64 `json:"image_route_failure_by_capability_profile"`
	ImageRouteLatencyMsByCapabilityProfile map[string]int64 `json:"image_route_latency_ms_by_capability_profile"`
	ImageRouteFailureByUpstreamStatus      map[string]int64 `json:"image_route_failure_by_upstream_status"`
	ResponsesImagegenCompatBySource        map[string]int64 `json:"responses_imagegen_compat_by_source"`
	ResponsesImagegenNormalizedBySource    map[string]int64 `json:"responses_imagegen_normalized_by_source"`
	ResponsesImagegenRejectByCode          map[string]int64 `json:"responses_imagegen_reject_by_code"`
	ResponsesImageToolFailureByProvider    map[string]int64 `json:"responses_image_tool_failure_by_provider"`
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
	publicModelProjectionTotal        atomic.Int64
	publicModelRestrictionHitTotal    atomic.Int64
	imageRouteTotal                   atomic.Int64
	imageRouteSuccessTotal            atomic.Int64
	imageRouteFailureTotal            atomic.Int64
	imageRouteLatencyMsTotal          atomic.Int64
	responsesImagegenCompatTotal      atomic.Int64
	responsesImagegenNormalizedTotal  atomic.Int64
	responsesImagegenRejectTotal      atomic.Int64
	responsesImageToolFailureTotal    atomic.Int64

	routeMismatchByKind                    sync.Map
	unsupportedActionByReason              sync.Map
	localizationFallbackByKind             sync.Map
	accountTestResolutionByReason          sync.Map
	accountProbeResolutionByReason         sync.Map
	recoveryProbeStartedByReason           sync.Map
	recoveryProbeSuccessByReason           sync.Map
	recoveryProbeRetryByReason             sync.Map
	recoveryProbeBlacklistedByReason       sync.Map
	billingResolverByPath                  sync.Map
	billingResolverFallbackByReason        sync.Map
	billingDeprecatedAPIByPath             sync.Map
	billingBulkApplyByOperation            sync.Map
	geminiBillingFallbackByReason          sync.Map
	geminiBillingFallbackMissByReason      sync.Map
	publicModelProjectionBySource          sync.Map
	publicModelRestrictionByReason         sync.Map
	imageRouteByFamily                     sync.Map
	imageRouteByProvider                   sync.Map
	imageRouteSuccessByFamily              sync.Map
	imageRouteSuccessByProvider            sync.Map
	imageRouteFailureByFamily              sync.Map
	imageRouteFailureByProvider            sync.Map
	imageRouteLatencyMsByFamily            sync.Map
	imageRouteLatencyMsByProvider          sync.Map
	imageRouteByProtocolMode               sync.Map
	imageRouteSuccessByProtocolMode        sync.Map
	imageRouteFailureByProtocolMode        sync.Map
	imageRouteLatencyMsByProtocolMode      sync.Map
	imageRouteByAction                     sync.Map
	imageRouteSuccessByAction              sync.Map
	imageRouteFailureByAction              sync.Map
	imageRouteLatencyMsByAction            sync.Map
	imageRouteBySizeTier                   sync.Map
	imageRouteSuccessBySizeTier            sync.Map
	imageRouteFailureBySizeTier            sync.Map
	imageRouteLatencyMsBySizeTier          sync.Map
	imageRouteByCapabilityProfile          sync.Map
	imageRouteSuccessByCapabilityProfile   sync.Map
	imageRouteFailureByCapabilityProfile   sync.Map
	imageRouteLatencyMsByCapabilityProfile sync.Map
	imageRouteFailureByUpstreamStatus      sync.Map
	responsesImagegenCompatBySource        sync.Map
	responsesImagegenNormalizedBySource    sync.Map
	responsesImagegenRejectByCode          sync.Map
	responsesImageToolFailureByProvider    sync.Map
}

var defaultMetrics metrics

func Snapshot() MetricsSnapshot {
	return MetricsSnapshot{
		RouteMismatchTotal:                     defaultMetrics.routeMismatchTotal.Load(),
		UnsupportedActionTotal:                 defaultMetrics.unsupportedActionTotal.Load(),
		LocalizationFallbackTotal:              defaultMetrics.localizationFallbackTotal.Load(),
		AccountTestResolutionFailedTotal:       defaultMetrics.accountTestResolutionFailedTotal.Load(),
		AccountProbeResolutionFailedTotal:      defaultMetrics.accountProbeResolutionFailedTotal.Load(),
		RecoveryProbeStartedTotal:              defaultMetrics.recoveryProbeStartedTotal.Load(),
		RecoveryProbeSuccessTotal:              defaultMetrics.recoveryProbeSuccessTotal.Load(),
		RecoveryProbeRetryTotal:                defaultMetrics.recoveryProbeRetryTotal.Load(),
		RecoveryProbeBlacklistedTotal:          defaultMetrics.recoveryProbeBlacklistedTotal.Load(),
		BillingResolverTotal:                   defaultMetrics.billingResolverTotal.Load(),
		BillingResolverFallbackTotal:           defaultMetrics.billingResolverFallbackTotal.Load(),
		BillingDeprecatedAPITotal:              defaultMetrics.billingDeprecatedAPITotal.Load(),
		BillingBulkApplyTotal:                  defaultMetrics.billingBulkApplyTotal.Load(),
		GeminiBillingFallbackAppliedTotal:      defaultMetrics.geminiBillingFallbackAppliedTotal.Load(),
		GeminiBillingFallbackMissTotal:         defaultMetrics.geminiBillingFallbackMissTotal.Load(),
		PublicModelProjectionTotal:             defaultMetrics.publicModelProjectionTotal.Load(),
		PublicModelRestrictionHitTotal:         defaultMetrics.publicModelRestrictionHitTotal.Load(),
		ImageRouteTotal:                        defaultMetrics.imageRouteTotal.Load(),
		ImageRouteSuccessTotal:                 defaultMetrics.imageRouteSuccessTotal.Load(),
		ImageRouteFailureTotal:                 defaultMetrics.imageRouteFailureTotal.Load(),
		ImageRouteLatencyMsTotal:               defaultMetrics.imageRouteLatencyMsTotal.Load(),
		ResponsesImagegenCompatTotal:           defaultMetrics.responsesImagegenCompatTotal.Load(),
		ResponsesImagegenNormalizedTotal:       defaultMetrics.responsesImagegenNormalizedTotal.Load(),
		ResponsesImagegenRejectTotal:           defaultMetrics.responsesImagegenRejectTotal.Load(),
		ResponsesImageToolFailureTotal:         defaultMetrics.responsesImageToolFailureTotal.Load(),
		RouteMismatchByKind:                    snapshotCounterMap(&defaultMetrics.routeMismatchByKind),
		UnsupportedActionByReason:              snapshotCounterMap(&defaultMetrics.unsupportedActionByReason),
		LocalizationFallbackByKind:             snapshotCounterMap(&defaultMetrics.localizationFallbackByKind),
		AccountTestResolutionByReason:          snapshotCounterMap(&defaultMetrics.accountTestResolutionByReason),
		AccountProbeResolutionByReason:         snapshotCounterMap(&defaultMetrics.accountProbeResolutionByReason),
		RecoveryProbeStartedByReason:           snapshotCounterMap(&defaultMetrics.recoveryProbeStartedByReason),
		RecoveryProbeSuccessByReason:           snapshotCounterMap(&defaultMetrics.recoveryProbeSuccessByReason),
		RecoveryProbeRetryByReason:             snapshotCounterMap(&defaultMetrics.recoveryProbeRetryByReason),
		RecoveryProbeBlacklistedByReason:       snapshotCounterMap(&defaultMetrics.recoveryProbeBlacklistedByReason),
		BillingResolverByPath:                  snapshotCounterMap(&defaultMetrics.billingResolverByPath),
		BillingResolverFallbackByReason:        snapshotCounterMap(&defaultMetrics.billingResolverFallbackByReason),
		BillingDeprecatedAPIByPath:             snapshotCounterMap(&defaultMetrics.billingDeprecatedAPIByPath),
		BillingBulkApplyByOperation:            snapshotCounterMap(&defaultMetrics.billingBulkApplyByOperation),
		GeminiBillingFallbackByReason:          snapshotCounterMap(&defaultMetrics.geminiBillingFallbackByReason),
		GeminiBillingFallbackMissByReason:      snapshotCounterMap(&defaultMetrics.geminiBillingFallbackMissByReason),
		PublicModelProjectionBySource:          snapshotCounterMap(&defaultMetrics.publicModelProjectionBySource),
		PublicModelRestrictionByReason:         snapshotCounterMap(&defaultMetrics.publicModelRestrictionByReason),
		ImageRouteByFamily:                     snapshotCounterMap(&defaultMetrics.imageRouteByFamily),
		ImageRouteByProvider:                   snapshotCounterMap(&defaultMetrics.imageRouteByProvider),
		ImageRouteSuccessByFamily:              snapshotCounterMap(&defaultMetrics.imageRouteSuccessByFamily),
		ImageRouteSuccessByProvider:            snapshotCounterMap(&defaultMetrics.imageRouteSuccessByProvider),
		ImageRouteFailureByFamily:              snapshotCounterMap(&defaultMetrics.imageRouteFailureByFamily),
		ImageRouteFailureByProvider:            snapshotCounterMap(&defaultMetrics.imageRouteFailureByProvider),
		ImageRouteLatencyMsByFamily:            snapshotCounterMap(&defaultMetrics.imageRouteLatencyMsByFamily),
		ImageRouteLatencyMsByProvider:          snapshotCounterMap(&defaultMetrics.imageRouteLatencyMsByProvider),
		ImageRouteByProtocolMode:               snapshotCounterMap(&defaultMetrics.imageRouteByProtocolMode),
		ImageRouteSuccessByProtocolMode:        snapshotCounterMap(&defaultMetrics.imageRouteSuccessByProtocolMode),
		ImageRouteFailureByProtocolMode:        snapshotCounterMap(&defaultMetrics.imageRouteFailureByProtocolMode),
		ImageRouteLatencyMsByProtocolMode:      snapshotCounterMap(&defaultMetrics.imageRouteLatencyMsByProtocolMode),
		ImageRouteByAction:                     snapshotCounterMap(&defaultMetrics.imageRouteByAction),
		ImageRouteSuccessByAction:              snapshotCounterMap(&defaultMetrics.imageRouteSuccessByAction),
		ImageRouteFailureByAction:              snapshotCounterMap(&defaultMetrics.imageRouteFailureByAction),
		ImageRouteLatencyMsByAction:            snapshotCounterMap(&defaultMetrics.imageRouteLatencyMsByAction),
		ImageRouteBySizeTier:                   snapshotCounterMap(&defaultMetrics.imageRouteBySizeTier),
		ImageRouteSuccessBySizeTier:            snapshotCounterMap(&defaultMetrics.imageRouteSuccessBySizeTier),
		ImageRouteFailureBySizeTier:            snapshotCounterMap(&defaultMetrics.imageRouteFailureBySizeTier),
		ImageRouteLatencyMsBySizeTier:          snapshotCounterMap(&defaultMetrics.imageRouteLatencyMsBySizeTier),
		ImageRouteByCapabilityProfile:          snapshotCounterMap(&defaultMetrics.imageRouteByCapabilityProfile),
		ImageRouteSuccessByCapabilityProfile:   snapshotCounterMap(&defaultMetrics.imageRouteSuccessByCapabilityProfile),
		ImageRouteFailureByCapabilityProfile:   snapshotCounterMap(&defaultMetrics.imageRouteFailureByCapabilityProfile),
		ImageRouteLatencyMsByCapabilityProfile: snapshotCounterMap(&defaultMetrics.imageRouteLatencyMsByCapabilityProfile),
		ImageRouteFailureByUpstreamStatus:      snapshotCounterMap(&defaultMetrics.imageRouteFailureByUpstreamStatus),
		ResponsesImagegenCompatBySource:        snapshotCounterMap(&defaultMetrics.responsesImagegenCompatBySource),
		ResponsesImagegenNormalizedBySource:    snapshotCounterMap(&defaultMetrics.responsesImagegenNormalizedBySource),
		ResponsesImagegenRejectByCode:          snapshotCounterMap(&defaultMetrics.responsesImagegenRejectByCode),
		ResponsesImageToolFailureByProvider:    snapshotCounterMap(&defaultMetrics.responsesImageToolFailureByProvider),
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

func RecordPublicModelProjection(source string) {
	defaultMetrics.publicModelProjectionTotal.Add(1)
	incrementCounterMap(&defaultMetrics.publicModelProjectionBySource, source)
}

func RecordPublicModelRestrictionHit(reason string) {
	defaultMetrics.publicModelRestrictionHitTotal.Add(1)
	incrementCounterMap(&defaultMetrics.publicModelRestrictionByReason, reason)
}

func RecordImageRoute(
	family string,
	provider string,
	protocolMode string,
	action string,
	sizeTier string,
	capabilityProfile string,
	success bool,
	latencyMs int64,
	upstreamStatus int,
) {
	defaultMetrics.imageRouteTotal.Add(1)
	incrementCounterMap(&defaultMetrics.imageRouteByFamily, family)
	incrementCounterMap(&defaultMetrics.imageRouteByProvider, provider)
	normalizedProtocolMode := normalizeImageRouteProtocolMode(protocolMode)
	normalizedAction := normalizeImageRouteAction(action)
	normalizedSizeTier := normalizeImageRouteSizeTier(sizeTier)
	normalizedCapabilityProfile := normalizeImageRouteCapabilityProfile(capabilityProfile)
	incrementCounterMap(&defaultMetrics.imageRouteByProtocolMode, normalizedProtocolMode)
	incrementCounterMap(&defaultMetrics.imageRouteByAction, normalizedAction)
	incrementCounterMap(&defaultMetrics.imageRouteBySizeTier, normalizedSizeTier)
	incrementCounterMap(&defaultMetrics.imageRouteByCapabilityProfile, normalizedCapabilityProfile)

	if success {
		defaultMetrics.imageRouteSuccessTotal.Add(1)
		incrementCounterMap(&defaultMetrics.imageRouteSuccessByFamily, family)
		incrementCounterMap(&defaultMetrics.imageRouteSuccessByProvider, provider)
		incrementCounterMap(&defaultMetrics.imageRouteSuccessByProtocolMode, normalizedProtocolMode)
		incrementCounterMap(&defaultMetrics.imageRouteSuccessByAction, normalizedAction)
		incrementCounterMap(&defaultMetrics.imageRouteSuccessBySizeTier, normalizedSizeTier)
		incrementCounterMap(&defaultMetrics.imageRouteSuccessByCapabilityProfile, normalizedCapabilityProfile)
	} else {
		defaultMetrics.imageRouteFailureTotal.Add(1)
		incrementCounterMap(&defaultMetrics.imageRouteFailureByFamily, family)
		incrementCounterMap(&defaultMetrics.imageRouteFailureByProvider, provider)
		incrementCounterMap(&defaultMetrics.imageRouteFailureByProtocolMode, normalizedProtocolMode)
		incrementCounterMap(&defaultMetrics.imageRouteFailureByAction, normalizedAction)
		incrementCounterMap(&defaultMetrics.imageRouteFailureBySizeTier, normalizedSizeTier)
		incrementCounterMap(&defaultMetrics.imageRouteFailureByCapabilityProfile, normalizedCapabilityProfile)
		if upstreamStatus > 0 {
			incrementCounterMap(&defaultMetrics.imageRouteFailureByUpstreamStatus, strconv.Itoa(upstreamStatus))
		}
	}

	if latencyMs < 0 {
		latencyMs = 0
	}
	defaultMetrics.imageRouteLatencyMsTotal.Add(latencyMs)
	addCounterMap(&defaultMetrics.imageRouteLatencyMsByFamily, family, latencyMs)
	addCounterMap(&defaultMetrics.imageRouteLatencyMsByProvider, provider, latencyMs)
	addCounterMap(&defaultMetrics.imageRouteLatencyMsByProtocolMode, normalizedProtocolMode, latencyMs)
	addCounterMap(&defaultMetrics.imageRouteLatencyMsByAction, normalizedAction, latencyMs)
	addCounterMap(&defaultMetrics.imageRouteLatencyMsBySizeTier, normalizedSizeTier, latencyMs)
	addCounterMap(&defaultMetrics.imageRouteLatencyMsByCapabilityProfile, normalizedCapabilityProfile, latencyMs)
}

func RecordResponsesImagegenCompat(source string) {
	defaultMetrics.responsesImagegenCompatTotal.Add(1)
	incrementCounterMap(&defaultMetrics.responsesImagegenCompatBySource, source)
}

func RecordResponsesImagegenNormalized(source string) {
	defaultMetrics.responsesImagegenNormalizedTotal.Add(1)
	incrementCounterMap(&defaultMetrics.responsesImagegenNormalizedBySource, source)
}

func RecordResponsesImagegenReject(code string) {
	defaultMetrics.responsesImagegenRejectTotal.Add(1)
	incrementCounterMap(&defaultMetrics.responsesImagegenRejectByCode, code)
}

func RecordResponsesImageToolFailure(provider string) {
	defaultMetrics.responsesImageToolFailureTotal.Add(1)
	incrementCounterMap(&defaultMetrics.responsesImageToolFailureByProvider, provider)
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
	defaultMetrics.publicModelProjectionTotal.Store(0)
	defaultMetrics.publicModelRestrictionHitTotal.Store(0)
	defaultMetrics.imageRouteTotal.Store(0)
	defaultMetrics.imageRouteSuccessTotal.Store(0)
	defaultMetrics.imageRouteFailureTotal.Store(0)
	defaultMetrics.imageRouteLatencyMsTotal.Store(0)
	defaultMetrics.responsesImagegenCompatTotal.Store(0)
	defaultMetrics.responsesImagegenNormalizedTotal.Store(0)
	defaultMetrics.responsesImagegenRejectTotal.Store(0)
	defaultMetrics.responsesImageToolFailureTotal.Store(0)
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
	resetCounterMap(&defaultMetrics.publicModelProjectionBySource)
	resetCounterMap(&defaultMetrics.publicModelRestrictionByReason)
	resetCounterMap(&defaultMetrics.imageRouteByFamily)
	resetCounterMap(&defaultMetrics.imageRouteByProvider)
	resetCounterMap(&defaultMetrics.imageRouteSuccessByFamily)
	resetCounterMap(&defaultMetrics.imageRouteSuccessByProvider)
	resetCounterMap(&defaultMetrics.imageRouteFailureByFamily)
	resetCounterMap(&defaultMetrics.imageRouteFailureByProvider)
	resetCounterMap(&defaultMetrics.imageRouteLatencyMsByFamily)
	resetCounterMap(&defaultMetrics.imageRouteLatencyMsByProvider)
	resetCounterMap(&defaultMetrics.imageRouteByProtocolMode)
	resetCounterMap(&defaultMetrics.imageRouteSuccessByProtocolMode)
	resetCounterMap(&defaultMetrics.imageRouteFailureByProtocolMode)
	resetCounterMap(&defaultMetrics.imageRouteLatencyMsByProtocolMode)
	resetCounterMap(&defaultMetrics.imageRouteByAction)
	resetCounterMap(&defaultMetrics.imageRouteSuccessByAction)
	resetCounterMap(&defaultMetrics.imageRouteFailureByAction)
	resetCounterMap(&defaultMetrics.imageRouteLatencyMsByAction)
	resetCounterMap(&defaultMetrics.imageRouteBySizeTier)
	resetCounterMap(&defaultMetrics.imageRouteSuccessBySizeTier)
	resetCounterMap(&defaultMetrics.imageRouteFailureBySizeTier)
	resetCounterMap(&defaultMetrics.imageRouteLatencyMsBySizeTier)
	resetCounterMap(&defaultMetrics.imageRouteByCapabilityProfile)
	resetCounterMap(&defaultMetrics.imageRouteSuccessByCapabilityProfile)
	resetCounterMap(&defaultMetrics.imageRouteFailureByCapabilityProfile)
	resetCounterMap(&defaultMetrics.imageRouteLatencyMsByCapabilityProfile)
	resetCounterMap(&defaultMetrics.imageRouteFailureByUpstreamStatus)
	resetCounterMap(&defaultMetrics.responsesImagegenCompatBySource)
	resetCounterMap(&defaultMetrics.responsesImagegenNormalizedBySource)
	resetCounterMap(&defaultMetrics.responsesImagegenRejectByCode)
	resetCounterMap(&defaultMetrics.responsesImageToolFailureByProvider)
}

func normalizeImageRouteProtocolMode(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "native":
		return "native"
	case "compat":
		return "compat"
	default:
		return "unknown"
	}
}

func normalizeImageRouteAction(value string) string {
	switch strings.TrimSpace(strings.ToLower(value)) {
	case "generate", "generation", "generations":
		return "generate"
	case "edit", "edits":
		return "edit"
	default:
		return "unknown"
	}
}

func normalizeImageRouteSizeTier(value string) string {
	switch strings.TrimSpace(strings.ToUpper(value)) {
	case "1K":
		return "1K"
	case "2K":
		return "2K"
	case "4K":
		return "4K"
	default:
		return "unknown"
	}
}

func normalizeImageRouteCapabilityProfile(value string) string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "unknown"
	}
	return trimmed
}

func incrementCounterMap(target *sync.Map, key string) {
	addCounterMap(target, key, 1)
}

func addCounterMap(target *sync.Map, key string, delta int64) {
	if delta == 0 {
		return
	}
	normalized := strings.TrimSpace(key)
	if normalized == "" {
		normalized = "unknown"
	}
	counter, _ := target.LoadOrStore(normalized, &atomic.Int64{})
	typedCounter, ok := counter.(*atomic.Int64)
	if !ok {
		return
	}
	typedCounter.Add(delta)
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
