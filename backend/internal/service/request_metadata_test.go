package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestRequestMetadataWriteAndRead_NoBridge(t *testing.T) {
	ctx := context.Background()
	ctx = WithIsMaxTokensOneHaikuRequest(ctx, true, false)
	ctx = WithThinkingEnabled(ctx, true, false)
	ctx = WithPrefetchedStickySession(ctx, 123, 456, false)
	ctx = WithSingleAccountRetry(ctx, true, false)
	ctx = WithAccountSwitchCount(ctx, 2, false)

	isHaiku, ok := IsMaxTokensOneHaikuRequestFromContext(ctx)
	require.True(t, ok)
	require.True(t, isHaiku)

	thinking, ok := ThinkingEnabledFromContext(ctx)
	require.True(t, ok)
	require.True(t, thinking)

	accountID, ok := PrefetchedStickyAccountIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(123), accountID)

	groupID, ok := PrefetchedStickyGroupIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(456), groupID)

	singleRetry, ok := SingleAccountRetryFromContext(ctx)
	require.True(t, ok)
	require.True(t, singleRetry)

	switchCount, ok := AccountSwitchCountFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, 2, switchCount)

	require.Nil(t, ctx.Value(ctxkey.IsMaxTokensOneHaikuRequest))
	require.Nil(t, ctx.Value(ctxkey.ThinkingEnabled))
	require.Nil(t, ctx.Value(ctxkey.PrefetchedStickyAccountID))
	require.Nil(t, ctx.Value(ctxkey.PrefetchedStickyGroupID))
	require.Nil(t, ctx.Value(ctxkey.SingleAccountRetry))
	require.Nil(t, ctx.Value(ctxkey.AccountSwitchCount))
}

func TestRequestMetadataWrite_BridgeLegacyKeys(t *testing.T) {
	ctx := context.Background()
	ctx = WithIsMaxTokensOneHaikuRequest(ctx, true, true)
	ctx = WithThinkingEnabled(ctx, true, true)
	ctx = WithPrefetchedStickySession(ctx, 123, 456, true)
	ctx = WithSingleAccountRetry(ctx, true, true)
	ctx = WithAccountSwitchCount(ctx, 2, true)

	require.Equal(t, true, ctx.Value(ctxkey.IsMaxTokensOneHaikuRequest))
	require.Equal(t, true, ctx.Value(ctxkey.ThinkingEnabled))
	require.Equal(t, int64(123), ctx.Value(ctxkey.PrefetchedStickyAccountID))
	require.Equal(t, int64(456), ctx.Value(ctxkey.PrefetchedStickyGroupID))
	require.Equal(t, true, ctx.Value(ctxkey.SingleAccountRetry))
	require.Equal(t, 2, ctx.Value(ctxkey.AccountSwitchCount))
}

func TestRequestMetadataRead_LegacyFallbackAndStats(t *testing.T) {
	beforeHaiku, beforeThinking, beforeAccount, beforeGroup, beforeSingleRetry, beforeSwitchCount := RequestMetadataFallbackStats()

	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkey.IsMaxTokensOneHaikuRequest, true)
	ctx = context.WithValue(ctx, ctxkey.ThinkingEnabled, true)
	ctx = context.WithValue(ctx, ctxkey.PrefetchedStickyAccountID, int64(321))
	ctx = context.WithValue(ctx, ctxkey.PrefetchedStickyGroupID, int64(654))
	ctx = context.WithValue(ctx, ctxkey.SingleAccountRetry, true)
	ctx = context.WithValue(ctx, ctxkey.AccountSwitchCount, int64(3))

	isHaiku, ok := IsMaxTokensOneHaikuRequestFromContext(ctx)
	require.True(t, ok)
	require.True(t, isHaiku)

	thinking, ok := ThinkingEnabledFromContext(ctx)
	require.True(t, ok)
	require.True(t, thinking)

	accountID, ok := PrefetchedStickyAccountIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(321), accountID)

	groupID, ok := PrefetchedStickyGroupIDFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, int64(654), groupID)

	singleRetry, ok := SingleAccountRetryFromContext(ctx)
	require.True(t, ok)
	require.True(t, singleRetry)

	switchCount, ok := AccountSwitchCountFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, 3, switchCount)

	afterHaiku, afterThinking, afterAccount, afterGroup, afterSingleRetry, afterSwitchCount := RequestMetadataFallbackStats()
	require.Equal(t, beforeHaiku+1, afterHaiku)
	require.Equal(t, beforeThinking+1, afterThinking)
	require.Equal(t, beforeAccount+1, afterAccount)
	require.Equal(t, beforeGroup+1, afterGroup)
	require.Equal(t, beforeSingleRetry+1, afterSingleRetry)
	require.Equal(t, beforeSwitchCount+1, afterSwitchCount)
}

func TestRequestMetadataRead_PreferMetadataOverLegacy(t *testing.T) {
	ctx := context.WithValue(context.Background(), ctxkey.ThinkingEnabled, false)
	ctx = WithThinkingEnabled(ctx, true, false)

	thinking, ok := ThinkingEnabledFromContext(ctx)
	require.True(t, ok)
	require.True(t, thinking)
	require.Equal(t, false, ctx.Value(ctxkey.ThinkingEnabled))
}

func TestRequestMetadataGeminiPublicFields(t *testing.T) {
	ctx := EnsureRequestMetadata(context.Background())
	SetGeminiPublicVersionMetadata(ctx, "v1alpha")
	SetGeminiPublicResourceMetadata(ctx, "live_auth_tokens")
	SetGeminiAliasUsedMetadata(ctx, true)
	SetGeminiModelMetadataSourceMetadata(ctx, "projected_empty")
	SetGeminiUpstreamPathMetadata(ctx, "/v1alpha/authTokens")

	version, ok := GeminiPublicVersionMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "v1alpha", version)

	resource, ok := GeminiPublicResourceMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "live_auth_tokens", resource)

	aliasUsed, ok := GeminiAliasUsedMetadataFromContext(ctx)
	require.True(t, ok)
	require.True(t, aliasUsed)

	metadataSource, ok := GeminiModelMetadataSourceMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "projected_empty", metadataSource)

	upstreamPath, ok := GeminiUpstreamPathMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "/v1alpha/authTokens", upstreamPath)
}

func TestRequestMetadataImageRouteFields(t *testing.T) {
	ctx := EnsureRequestMetadata(context.Background())
	SetImageRouteFamilyMetadata(ctx, PublicImageRouteFamily)
	SetImageActionMetadata(ctx, "generations")
	SetImageResolvedProviderMetadata(ctx, PlatformOpenAI)
	SetImageDisplayModelIDMetadata(ctx, "gpt-image-2")
	SetImageTargetModelIDMetadata(ctx, "gpt-image-2")
	SetImageUpstreamEndpointMetadata(ctx, EndpointImagesGen)
	SetImageRequestFormatMetadata(ctx, "application/json")
	SetImageRouteReasonMetadata(ctx, PublicImageRouteReasonModelProvider)
	SetImageProtocolModeMetadata(ctx, OpenAIImageProtocolModeCompat)
	SetImageRequestSurfaceMetadata(ctx, "images_bridge")
	SetImageSizeTierMetadata(ctx, OpenAIImageSizeTier2K)
	SetImageCapabilityProfileMetadata(ctx, "openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on")
	SetImageOutputCountMetadata(ctx, 2)

	routeFamily, ok := ImageRouteFamilyMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, PublicImageRouteFamily, routeFamily)

	action, ok := ImageActionMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "generations", action)

	provider, ok := ImageResolvedProviderMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, PlatformOpenAI, provider)

	displayModelID, ok := ImageDisplayModelIDMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "gpt-image-2", displayModelID)

	targetModelID, ok := ImageTargetModelIDMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "gpt-image-2", targetModelID)

	upstreamEndpoint, ok := ImageUpstreamEndpointMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, EndpointImagesGen, upstreamEndpoint)

	requestFormat, ok := ImageRequestFormatMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "application/json", requestFormat)

	routeReason, ok := ImageRouteReasonMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, PublicImageRouteReasonModelProvider, routeReason)

	protocolMode, ok := ImageProtocolModeMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, OpenAIImageProtocolModeCompat, protocolMode)

	requestSurface, ok := ImageRequestSurfaceMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "images_bridge", requestSurface)

	sizeTier, ok := ImageSizeTierMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, OpenAIImageSizeTier2K, sizeTier)

	capabilityProfile, ok := ImageCapabilityProfileMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "openai_image.compat.gpt-image-2.transparent_on.custom_resolution_on", capabilityProfile)

	outputCount, ok := ImageOutputCountMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, 2, outputCount)
}

func TestRequestMetadataOpenAIResponsesImagegenCompatFields(t *testing.T) {
	ctx := EnsureRequestMetadata(context.Background())
	SetOpenAIResponsesImageGenCompatMetadata(ctx, OpenAIResponsesCompatMetadata{
		Enabled:                   true,
		Rejected:                  true,
		RejectCode:                "multipart_stream_unsupported",
		SourceGuess:               OpenAIResponsesImagegenCompatSourceMultipart,
		Source:                    OpenAIResponsesImagegenCompatSourceMultipart,
		ReferenceImageCount:       2,
		ReferenceImageBytesBefore: 4096,
		ReferenceImageBytesAfter:  2048,
		ReferenceImagesNormalized: true,
		ImageGenerationSize:       "1536x1024",
	})

	metadata, ok := OpenAIResponsesImageGenCompatMetadataFromContext(ctx)
	require.True(t, ok)
	require.True(t, metadata.Enabled)
	require.True(t, metadata.Rejected)
	require.Equal(t, "multipart_stream_unsupported", metadata.RejectCode)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceMultipart, metadata.SourceGuess)
	require.Equal(t, OpenAIResponsesImagegenCompatSourceMultipart, metadata.Source)
	require.Equal(t, 2, metadata.ReferenceImageCount)
	require.Equal(t, int64(4096), metadata.ReferenceImageBytesBefore)
	require.Equal(t, int64(2048), metadata.ReferenceImageBytesAfter)
	require.True(t, metadata.ReferenceImagesNormalized)
	require.Equal(t, "1536x1024", metadata.ImageGenerationSize)
}
