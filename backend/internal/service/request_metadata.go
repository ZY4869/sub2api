package service

import (
	"context"
	"strings"
	"sync/atomic"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
)

type requestMetadataContextKey struct{}

var requestMetadataKey = requestMetadataContextKey{}

type RequestMetadata struct {
	IsMaxTokensOneHaikuRequest  *bool
	ThinkingEnabled             *bool
	PrefetchedStickyAccountID   *int64
	PrefetchedStickyGroupID     *int64
	SingleAccountRetry          *bool
	AccountSwitchCount          *int
	GeminiSurface               *string
	GeminiRequestedServiceTier  *string
	GeminiResolvedServiceTier   *string
	GeminiBatchMode             *string
	GeminiCachePhase            *string
	GeminiPublicVersion         *string
	GeminiPublicResource        *string
	GeminiAliasUsed             *bool
	GeminiModelMetadataSource   *string
	GeminiUpstreamPath          *string
	GeminiBillingFallbackReason *string
	BillingRuleID               *string
	ProbeAction                 *string
	ImageRouteFamily            *string
	ImageAction                 *string
	ImageResolvedProvider       *string
	ImageDisplayModelID         *string
	ImageTargetModelID          *string
	ImageUpstreamEndpoint       *string
	ImageRequestFormat          *string
	ImageRouteReason            *string
	ImageProtocolMode           *string
	ImageRequestSurface         *string
	ImageSizeTier               *string
	ImageCapabilityProfile      *string
	ImageOutputCount            *int
	ImagegenCompat              *bool
	ImagegenCompatRejected      *bool
	ImagegenCompatRejectCode    *string
	ImagegenCompatSourceGuess   *string
	ImagegenCompatSource        *string
	ImagegenCompatRefCount      *int
	ImagegenCompatBytesBefore   *int64
	ImagegenCompatBytesAfter    *int64
	ImagegenCompatNormalized    *bool
	ImagegenCompatSize          *string
}

var (
	requestMetadataFallbackIsMaxTokensOneHaikuTotal atomic.Int64
	requestMetadataFallbackThinkingEnabledTotal     atomic.Int64
	requestMetadataFallbackPrefetchedStickyAccount  atomic.Int64
	requestMetadataFallbackPrefetchedStickyGroup    atomic.Int64
	requestMetadataFallbackSingleAccountRetryTotal  atomic.Int64
	requestMetadataFallbackAccountSwitchCountTotal  atomic.Int64
)

func RequestMetadataFallbackStats() (isMaxTokensOneHaiku, thinkingEnabled, prefetchedStickyAccount, prefetchedStickyGroup, singleAccountRetry, accountSwitchCount int64) {
	return requestMetadataFallbackIsMaxTokensOneHaikuTotal.Load(),
		requestMetadataFallbackThinkingEnabledTotal.Load(),
		requestMetadataFallbackPrefetchedStickyAccount.Load(),
		requestMetadataFallbackPrefetchedStickyGroup.Load(),
		requestMetadataFallbackSingleAccountRetryTotal.Load(),
		requestMetadataFallbackAccountSwitchCountTotal.Load()
}

func metadataFromContext(ctx context.Context) *RequestMetadata {
	if ctx == nil {
		return nil
	}
	md, _ := ctx.Value(requestMetadataKey).(*RequestMetadata)
	return md
}

func EnsureRequestMetadata(ctx context.Context) context.Context {
	if ctx == nil {
		return nil
	}
	if metadataFromContext(ctx) != nil {
		return ctx
	}
	return context.WithValue(ctx, requestMetadataKey, &RequestMetadata{})
}

func updateRequestMetadata(
	ctx context.Context,
	bridgeOldKeys bool,
	update func(md *RequestMetadata),
	legacyBridge func(ctx context.Context) context.Context,
) context.Context {
	if ctx == nil {
		return nil
	}
	current := metadataFromContext(ctx)
	next := &RequestMetadata{}
	if current != nil {
		*next = *current
	}
	update(next)
	ctx = context.WithValue(ctx, requestMetadataKey, next)
	if bridgeOldKeys && legacyBridge != nil {
		ctx = legacyBridge(ctx)
	}
	return ctx
}

func WithIsMaxTokensOneHaikuRequest(ctx context.Context, value bool, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.IsMaxTokensOneHaikuRequest = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.IsMaxTokensOneHaikuRequest, value)
	})
}

func WithThinkingEnabled(ctx context.Context, value bool, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.ThinkingEnabled = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.ThinkingEnabled, value)
	})
}

func WithPrefetchedStickySession(ctx context.Context, accountID, groupID int64, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		account := accountID
		group := groupID
		md.PrefetchedStickyAccountID = &account
		md.PrefetchedStickyGroupID = &group
	}, func(base context.Context) context.Context {
		bridged := context.WithValue(base, ctxkey.PrefetchedStickyAccountID, accountID)
		return context.WithValue(bridged, ctxkey.PrefetchedStickyGroupID, groupID)
	})
}

func WithSingleAccountRetry(ctx context.Context, value bool, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.SingleAccountRetry = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.SingleAccountRetry, value)
	})
}

func WithAccountSwitchCount(ctx context.Context, value int, bridgeOldKeys bool) context.Context {
	return updateRequestMetadata(ctx, bridgeOldKeys, func(md *RequestMetadata) {
		v := value
		md.AccountSwitchCount = &v
	}, func(base context.Context) context.Context {
		return context.WithValue(base, ctxkey.AccountSwitchCount, value)
	})
}

func IsMaxTokensOneHaikuRequestFromContext(ctx context.Context) (bool, bool) {
	if md := metadataFromContext(ctx); md != nil && md.IsMaxTokensOneHaikuRequest != nil {
		return *md.IsMaxTokensOneHaikuRequest, true
	}
	if ctx == nil {
		return false, false
	}
	if value, ok := ctx.Value(ctxkey.IsMaxTokensOneHaikuRequest).(bool); ok {
		requestMetadataFallbackIsMaxTokensOneHaikuTotal.Add(1)
		return value, true
	}
	return false, false
}

func ThinkingEnabledFromContext(ctx context.Context) (bool, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ThinkingEnabled != nil {
		return *md.ThinkingEnabled, true
	}
	if ctx == nil {
		return false, false
	}
	if value, ok := ctx.Value(ctxkey.ThinkingEnabled).(bool); ok {
		requestMetadataFallbackThinkingEnabledTotal.Add(1)
		return value, true
	}
	return false, false
}

func PrefetchedStickyGroupIDFromContext(ctx context.Context) (int64, bool) {
	if md := metadataFromContext(ctx); md != nil && md.PrefetchedStickyGroupID != nil {
		return *md.PrefetchedStickyGroupID, true
	}
	if ctx == nil {
		return 0, false
	}
	v := ctx.Value(ctxkey.PrefetchedStickyGroupID)
	switch t := v.(type) {
	case int64:
		requestMetadataFallbackPrefetchedStickyGroup.Add(1)
		return t, true
	case int:
		requestMetadataFallbackPrefetchedStickyGroup.Add(1)
		return int64(t), true
	}
	return 0, false
}

func PrefetchedStickyAccountIDFromContext(ctx context.Context) (int64, bool) {
	if md := metadataFromContext(ctx); md != nil && md.PrefetchedStickyAccountID != nil {
		return *md.PrefetchedStickyAccountID, true
	}
	if ctx == nil {
		return 0, false
	}
	v := ctx.Value(ctxkey.PrefetchedStickyAccountID)
	switch t := v.(type) {
	case int64:
		requestMetadataFallbackPrefetchedStickyAccount.Add(1)
		return t, true
	case int:
		requestMetadataFallbackPrefetchedStickyAccount.Add(1)
		return int64(t), true
	}
	return 0, false
}

func SingleAccountRetryFromContext(ctx context.Context) (bool, bool) {
	if md := metadataFromContext(ctx); md != nil && md.SingleAccountRetry != nil {
		return *md.SingleAccountRetry, true
	}
	if ctx == nil {
		return false, false
	}
	if value, ok := ctx.Value(ctxkey.SingleAccountRetry).(bool); ok {
		requestMetadataFallbackSingleAccountRetryTotal.Add(1)
		return value, true
	}
	return false, false
}

func AccountSwitchCountFromContext(ctx context.Context) (int, bool) {
	if md := metadataFromContext(ctx); md != nil && md.AccountSwitchCount != nil {
		return *md.AccountSwitchCount, true
	}
	if ctx == nil {
		return 0, false
	}
	v := ctx.Value(ctxkey.AccountSwitchCount)
	switch t := v.(type) {
	case int:
		requestMetadataFallbackAccountSwitchCountTotal.Add(1)
		return t, true
	case int64:
		requestMetadataFallbackAccountSwitchCountTotal.Add(1)
		return int(t), true
	}
	return 0, false
}

func SetGeminiSurfaceMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiSurface = nil
			return
		}
		md.GeminiSurface = &trimmed
	}
}

func GeminiSurfaceMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiSurface != nil {
		return strings.TrimSpace(*md.GeminiSurface), true
	}
	return "", false
}

func SetGeminiRequestedServiceTierMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiRequestedServiceTier = nil
			return
		}
		md.GeminiRequestedServiceTier = &trimmed
	}
}

func GeminiRequestedServiceTierMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiRequestedServiceTier != nil {
		return strings.TrimSpace(*md.GeminiRequestedServiceTier), true
	}
	return "", false
}

func SetGeminiResolvedServiceTierMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiResolvedServiceTier = nil
			return
		}
		md.GeminiResolvedServiceTier = &trimmed
	}
}

func GeminiResolvedServiceTierMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiResolvedServiceTier != nil {
		return strings.TrimSpace(*md.GeminiResolvedServiceTier), true
	}
	return "", false
}

func SetGeminiBatchModeMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiBatchMode = nil
			return
		}
		md.GeminiBatchMode = &trimmed
	}
}

func GeminiBatchModeMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiBatchMode != nil {
		return strings.TrimSpace(*md.GeminiBatchMode), true
	}
	return "", false
}

func SetGeminiCachePhaseMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiCachePhase = nil
			return
		}
		md.GeminiCachePhase = &trimmed
	}
}

func GeminiCachePhaseMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiCachePhase != nil {
		return strings.TrimSpace(*md.GeminiCachePhase), true
	}
	return "", false
}

func SetGeminiPublicVersionMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiPublicVersion = nil
			return
		}
		md.GeminiPublicVersion = &trimmed
	}
}

func GeminiPublicVersionMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiPublicVersion != nil {
		return strings.TrimSpace(*md.GeminiPublicVersion), true
	}
	return "", false
}

func SetGeminiPublicResourceMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiPublicResource = nil
			return
		}
		md.GeminiPublicResource = &trimmed
	}
}

func GeminiPublicResourceMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiPublicResource != nil {
		return strings.TrimSpace(*md.GeminiPublicResource), true
	}
	return "", false
}

func SetGeminiAliasUsedMetadata(ctx context.Context, value bool) {
	if md := metadataFromContext(ctx); md != nil {
		v := value
		md.GeminiAliasUsed = &v
	}
}

func GeminiAliasUsedMetadataFromContext(ctx context.Context) (bool, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiAliasUsed != nil {
		return *md.GeminiAliasUsed, true
	}
	return false, false
}

func SetGeminiModelMetadataSourceMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiModelMetadataSource = nil
			return
		}
		md.GeminiModelMetadataSource = &trimmed
	}
}

func GeminiModelMetadataSourceMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiModelMetadataSource != nil {
		return strings.TrimSpace(*md.GeminiModelMetadataSource), true
	}
	return "", false
}

func SetGeminiUpstreamPathMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiUpstreamPath = nil
			return
		}
		md.GeminiUpstreamPath = &trimmed
	}
}

func GeminiUpstreamPathMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiUpstreamPath != nil {
		return strings.TrimSpace(*md.GeminiUpstreamPath), true
	}
	return "", false
}

func SetGeminiBillingFallbackReasonMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.GeminiBillingFallbackReason = nil
			return
		}
		md.GeminiBillingFallbackReason = &trimmed
	}
}

func GeminiBillingFallbackReasonMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.GeminiBillingFallbackReason != nil {
		return strings.TrimSpace(*md.GeminiBillingFallbackReason), true
	}
	return "", false
}

func SetBillingRuleIDMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.BillingRuleID = nil
			return
		}
		md.BillingRuleID = &trimmed
	}
}

func BillingRuleIDMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.BillingRuleID != nil {
		return strings.TrimSpace(*md.BillingRuleID), true
	}
	return "", false
}

func SetProbeActionMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			md.ProbeAction = nil
			return
		}
		md.ProbeAction = &trimmed
	}
}

func ProbeActionMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ProbeAction != nil {
		return strings.TrimSpace(*md.ProbeAction), true
	}
	return "", false
}

func setTrimmedMetadataField(target **string, value string) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		*target = nil
		return
	}
	*target = &trimmed
}

func setBoolMetadataField(target **bool, value bool) {
	v := value
	*target = &v
}

func setIntMetadataField(target **int, value int) {
	v := value
	*target = &v
}

func setInt64MetadataField(target **int64, value int64) {
	v := value
	*target = &v
}

func SetImageRouteFamilyMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageRouteFamily, value)
	}
}

func ImageRouteFamilyMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageRouteFamily != nil {
		return strings.TrimSpace(*md.ImageRouteFamily), true
	}
	return "", false
}

func SetImageActionMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageAction, value)
	}
}

func ImageActionMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageAction != nil {
		return strings.TrimSpace(*md.ImageAction), true
	}
	return "", false
}

func SetImageResolvedProviderMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageResolvedProvider, value)
	}
}

func ImageResolvedProviderMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageResolvedProvider != nil {
		return strings.TrimSpace(*md.ImageResolvedProvider), true
	}
	return "", false
}

func SetImageDisplayModelIDMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageDisplayModelID, value)
	}
}

func ImageDisplayModelIDMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageDisplayModelID != nil {
		return strings.TrimSpace(*md.ImageDisplayModelID), true
	}
	return "", false
}

func SetImageTargetModelIDMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageTargetModelID, value)
	}
}

func ImageTargetModelIDMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageTargetModelID != nil {
		return strings.TrimSpace(*md.ImageTargetModelID), true
	}
	return "", false
}

func SetImageUpstreamEndpointMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageUpstreamEndpoint, value)
	}
}

func ImageUpstreamEndpointMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageUpstreamEndpoint != nil {
		return strings.TrimSpace(*md.ImageUpstreamEndpoint), true
	}
	return "", false
}

func SetImageRequestFormatMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageRequestFormat, value)
	}
}

func ImageRequestFormatMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageRequestFormat != nil {
		return strings.TrimSpace(*md.ImageRequestFormat), true
	}
	return "", false
}

func SetImageRouteReasonMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageRouteReason, value)
	}
}

func ImageRouteReasonMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageRouteReason != nil {
		return strings.TrimSpace(*md.ImageRouteReason), true
	}
	return "", false
}

func SetImageProtocolModeMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageProtocolMode, value)
	}
}

func ImageProtocolModeMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageProtocolMode != nil {
		return strings.TrimSpace(*md.ImageProtocolMode), true
	}
	return "", false
}

func SetImageRequestSurfaceMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageRequestSurface, value)
	}
}

func ImageRequestSurfaceMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageRequestSurface != nil {
		return strings.TrimSpace(*md.ImageRequestSurface), true
	}
	return "", false
}

func SetImageSizeTierMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageSizeTier, value)
	}
}

func ImageSizeTierMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageSizeTier != nil {
		return strings.TrimSpace(*md.ImageSizeTier), true
	}
	return "", false
}

func SetImageCapabilityProfileMetadata(ctx context.Context, value string) {
	if md := metadataFromContext(ctx); md != nil {
		setTrimmedMetadataField(&md.ImageCapabilityProfile, value)
	}
}

func ImageCapabilityProfileMetadataFromContext(ctx context.Context) (string, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageCapabilityProfile != nil {
		return strings.TrimSpace(*md.ImageCapabilityProfile), true
	}
	return "", false
}

func SetImageOutputCountMetadata(ctx context.Context, value int) {
	if md := metadataFromContext(ctx); md != nil {
		setIntMetadataField(&md.ImageOutputCount, value)
	}
}

func ImageOutputCountMetadataFromContext(ctx context.Context) (int, bool) {
	if md := metadataFromContext(ctx); md != nil && md.ImageOutputCount != nil {
		return *md.ImageOutputCount, true
	}
	return 0, false
}

func SetOpenAIResponsesImageGenCompatMetadata(ctx context.Context, value OpenAIResponsesCompatMetadata) {
	if md := metadataFromContext(ctx); md != nil {
		setBoolMetadataField(&md.ImagegenCompat, value.Enabled)
		setBoolMetadataField(&md.ImagegenCompatRejected, value.Rejected)
		setTrimmedMetadataField(&md.ImagegenCompatRejectCode, value.RejectCode)
		setTrimmedMetadataField(&md.ImagegenCompatSourceGuess, value.SourceGuess)
		setTrimmedMetadataField(&md.ImagegenCompatSource, value.Source)
		setIntMetadataField(&md.ImagegenCompatRefCount, value.ReferenceImageCount)
		setInt64MetadataField(&md.ImagegenCompatBytesBefore, value.ReferenceImageBytesBefore)
		setInt64MetadataField(&md.ImagegenCompatBytesAfter, value.ReferenceImageBytesAfter)
		setBoolMetadataField(&md.ImagegenCompatNormalized, value.ReferenceImagesNormalized)
		setTrimmedMetadataField(&md.ImagegenCompatSize, value.ImageGenerationSize)
	}
}

func OpenAIResponsesImageGenCompatMetadataFromContext(ctx context.Context) (OpenAIResponsesCompatMetadata, bool) {
	md := metadataFromContext(ctx)
	if md == nil || md.ImagegenCompat == nil {
		return OpenAIResponsesCompatMetadata{}, false
	}
	result := OpenAIResponsesCompatMetadata{
		Enabled: *md.ImagegenCompat,
	}
	if md.ImagegenCompatRejected != nil {
		result.Rejected = *md.ImagegenCompatRejected
	}
	if md.ImagegenCompatRejectCode != nil {
		result.RejectCode = strings.TrimSpace(*md.ImagegenCompatRejectCode)
	}
	if md.ImagegenCompatSourceGuess != nil {
		result.SourceGuess = strings.TrimSpace(*md.ImagegenCompatSourceGuess)
	}
	if md.ImagegenCompatSource != nil {
		result.Source = strings.TrimSpace(*md.ImagegenCompatSource)
	}
	if md.ImagegenCompatRefCount != nil {
		result.ReferenceImageCount = *md.ImagegenCompatRefCount
	}
	if md.ImagegenCompatBytesBefore != nil {
		result.ReferenceImageBytesBefore = *md.ImagegenCompatBytesBefore
	}
	if md.ImagegenCompatBytesAfter != nil {
		result.ReferenceImageBytesAfter = *md.ImagegenCompatBytesAfter
	}
	if md.ImagegenCompatNormalized != nil {
		result.ReferenceImagesNormalized = *md.ImagegenCompatNormalized
	}
	if md.ImagegenCompatSize != nil {
		result.ImageGenerationSize = strings.TrimSpace(*md.ImagegenCompatSize)
	}
	return result, true
}
