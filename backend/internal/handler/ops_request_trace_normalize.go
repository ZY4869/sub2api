package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type opsTraceUsage struct {
	inputTokens  int
	outputTokens int
	totalTokens  int
}

func buildOpsTraceNormalizeResult(c *gin.Context, apiKey *service.APIKey, requestBody []byte, responseBody []byte) (service.ProtocolNormalizeResult, opsTraceUsage) {
	result := service.ProtocolNormalizeResult{}
	usage := opsTraceUsage{}

	requestJSON := parseOpsTraceJSON(requestBody)
	responseJSON := parseOpsTraceResponseJSON(responseBody)

	result.Platform = resolveOpsTracePlatform(c, apiKey)
	result.ProtocolIn = inferOpsTraceProtocolIn(c)
	result.ProtocolOut = inferOpsTraceProtocolOut(c, result.ProtocolIn, result.Platform)
	result.Channel = inferOpsTraceChannel(c, result.ProtocolIn, result.ProtocolOut)
	result.RoutePath = inferOpsTraceRoutePath(c)
	result.RequestType = inferOpsTraceRequestType(c)
	result.RequestedModel = strings.TrimSpace(readOpsTraceModel(c))
	result.UpstreamModel = strings.TrimSpace(resolveOpsUpstreamModel(c))
	if result.UpstreamModel == "" {
		result.UpstreamModel = result.RequestedModel
	}

	if requestJSON != nil {
		enrichOpsTraceRequestMetadata(c.Request.Context(), requestJSON, &result)
	}
	if responseJSON != nil {
		enrichOpsTraceResponseMetadata(responseJSON, result.ProtocolOut, &result, &usage)
	}

	if result.ActualUpstreamModel == "" {
		result.ActualUpstreamModel = firstNonEmptyString(result.UpstreamModel, result.RequestedModel)
	}
	if result.UpstreamRequestID == "" {
		result.UpstreamRequestID = strings.TrimSpace(firstNonEmptyString(
			c.Writer.Header().Get("X-Request-Id"),
			c.Writer.Header().Get("x-request-id"),
			c.Writer.Header().Get("x-goog-request-id"),
		))
	}
	if geminiSurface, ok := service.GeminiSurfaceMetadataFromContext(c.Request.Context()); ok {
		result.GeminiSurface = geminiSurface
	} else if result.Platform == service.PlatformGemini {
		classifier := service.NewGeminiRequestClassifier()
		classification := classifier.ClassifyRequest(service.GeminiBillingCalculationInput{
			InboundEndpoint: c.Request.URL.Path,
			RequestBody:     requestBody,
		})
		if classification != nil {
			result.GeminiSurface = classification.Surface
			result.GeminiRequestedServiceTier = classification.RequestedServiceTier
			result.GeminiResolvedServiceTier = classification.ServiceTier
			result.GeminiBatchMode = classification.BatchMode
			result.GeminiCachePhase = classification.CachePhase
		}
	}
	if geminiRequestedServiceTier, ok := service.GeminiRequestedServiceTierMetadataFromContext(c.Request.Context()); ok {
		result.GeminiRequestedServiceTier = geminiRequestedServiceTier
	}
	if geminiResolvedServiceTier, ok := service.GeminiResolvedServiceTierMetadataFromContext(c.Request.Context()); ok {
		result.GeminiResolvedServiceTier = geminiResolvedServiceTier
	}
	if geminiBatchMode, ok := service.GeminiBatchModeMetadataFromContext(c.Request.Context()); ok {
		result.GeminiBatchMode = geminiBatchMode
	}
	if geminiCachePhase, ok := service.GeminiCachePhaseMetadataFromContext(c.Request.Context()); ok {
		result.GeminiCachePhase = geminiCachePhase
	}
	if geminiPublicVersion, ok := service.GeminiPublicVersionMetadataFromContext(c.Request.Context()); ok {
		result.GeminiPublicVersion = geminiPublicVersion
	}
	if geminiPublicResource, ok := service.GeminiPublicResourceMetadataFromContext(c.Request.Context()); ok {
		result.GeminiPublicResource = geminiPublicResource
	}
	if geminiAliasUsed, ok := service.GeminiAliasUsedMetadataFromContext(c.Request.Context()); ok {
		result.GeminiAliasUsed = geminiAliasUsed
	}
	if metadataSource, ok := service.GeminiModelMetadataSourceMetadataFromContext(c.Request.Context()); ok {
		result.GeminiModelMetadataSource = metadataSource
	}
	if upstreamPath, ok := service.GeminiUpstreamPathMetadataFromContext(c.Request.Context()); ok {
		result.UpstreamPath = upstreamPath
	} else {
		result.UpstreamPath = strings.TrimSpace(resolveOpsUpstreamEndpoint(c, result.Platform))
	}
	if billingRuleID, ok := service.BillingRuleIDMetadataFromContext(c.Request.Context()); ok {
		result.BillingRuleID = billingRuleID
	}
	if fallbackReason, ok := service.GeminiBillingFallbackReasonMetadataFromContext(c.Request.Context()); ok {
		result.GeminiBillingFallbackReason = fallbackReason
	}
	if probeAction, ok := service.ProbeActionMetadataFromContext(c.Request.Context()); ok {
		result.ProbeAction = probeAction
	}
	if imageRouteFamily, ok := service.ImageRouteFamilyMetadataFromContext(c.Request.Context()); ok {
		result.ImageRouteFamily = imageRouteFamily
	}
	if imageAction, ok := service.ImageActionMetadataFromContext(c.Request.Context()); ok {
		result.ImageAction = imageAction
	}
	if imageResolvedProvider, ok := service.ImageResolvedProviderMetadataFromContext(c.Request.Context()); ok {
		result.ImageResolvedProvider = imageResolvedProvider
	}
	if imageDisplayModelID, ok := service.ImageDisplayModelIDMetadataFromContext(c.Request.Context()); ok {
		result.ImageDisplayModelID = imageDisplayModelID
	}
	if imageTargetModelID, ok := service.ImageTargetModelIDMetadataFromContext(c.Request.Context()); ok {
		result.ImageTargetModelID = imageTargetModelID
	}
	if imageUpstreamEndpoint, ok := service.ImageUpstreamEndpointMetadataFromContext(c.Request.Context()); ok {
		result.ImageUpstreamEndpoint = imageUpstreamEndpoint
	}
	if imageRequestFormat, ok := service.ImageRequestFormatMetadataFromContext(c.Request.Context()); ok {
		result.ImageRequestFormat = imageRequestFormat
	}
	if imageRouteReason, ok := service.ImageRouteReasonMetadataFromContext(c.Request.Context()); ok {
		result.ImageRouteReason = imageRouteReason
	}
	if imageProtocolMode, ok := service.ImageProtocolModeMetadataFromContext(c.Request.Context()); ok {
		result.ImageProtocolMode = imageProtocolMode
	}
	if imageRequestSurface, ok := service.ImageRequestSurfaceMetadataFromContext(c.Request.Context()); ok {
		result.ImageRequestSurface = imageRequestSurface
	}
	if imageSizeTier, ok := service.ImageSizeTierMetadataFromContext(c.Request.Context()); ok {
		result.ImageSizeTier = imageSizeTier
	}
	if imageCapabilityProfile, ok := service.ImageCapabilityProfileMetadataFromContext(c.Request.Context()); ok {
		result.ImageCapabilityProfile = imageCapabilityProfile
	}
	if compatMetadata, ok := service.OpenAIResponsesImageGenCompatMetadataFromContext(c.Request.Context()); ok {
		result.ImagegenCompat = compatMetadata.Enabled
		result.ImagegenCompatRejected = compatMetadata.Rejected
		result.ImagegenCompatRejectCode = compatMetadata.RejectCode
		result.ImagegenCompatSourceGuess = compatMetadata.SourceGuess
		result.ImagegenCompatSource = compatMetadata.Source
		result.ImagegenCompatReferenceImageCount = compatMetadata.ReferenceImageCount
		result.ImagegenCompatReferenceImageBytesBefore = compatMetadata.ReferenceImageBytesBefore
		result.ImagegenCompatReferenceImageBytesAfter = compatMetadata.ReferenceImageBytesAfter
		result.ImagegenCompatNormalized = compatMetadata.ReferenceImagesNormalized
		result.ImagegenCompatImageGenerationSize = compatMetadata.ImageGenerationSize
		if result.MediaResolution == "" && strings.TrimSpace(compatMetadata.ImageGenerationSize) != "" {
			result.MediaResolution = strings.TrimSpace(compatMetadata.ImageGenerationSize)
		}
	}
	if headerValue := strings.TrimSpace(c.Writer.Header().Get("X-Sub2api-CountTokens-Source")); headerValue != "" {
		result.CountTokensSource = headerValue
	}
	return result, usage
}
