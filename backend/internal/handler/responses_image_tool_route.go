package handler

import (
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func detectResponsesImageToolRequest(body []byte) (string, bool) {
	return service.DetectOpenAIResponsesImageGenerationToolModel(body)
}

func supportsResponsesImageToolPlatform(platform string) bool {
	return service.IsOpenAIFamily(platform)
}

func responsesImageToolResolvedProvider(platform string) string {
	switch service.NormalizePlatformFamily(platform) {
	case service.PlatformOpenAI:
		return service.PlatformOpenAI
	case service.PlatformGrok:
		return service.PlatformGrok
	case service.PlatformGemini:
		return service.PlatformGemini
	default:
		return strings.TrimSpace(strings.ToLower(platform))
	}
}

func responsesImageToolUnsupportedPlatformMessage() string {
	return "image_generation tool on /v1/responses is only available for OpenAI/Codex accounts; use /grok/v1/images/* for Grok or /v1beta/openai/images/generations for Gemini native image models"
}

func applyResponsesImageToolTraceMetadata(
	c *gin.Context,
	platform string,
	requestedModel string,
	toolModel string,
	routeReason string,
) {
	if c == nil || c.Request == nil {
		return
	}
	if strings.TrimSpace(routeReason) == "" {
		routeReason = service.PublicImageToolRouteReason
	}

	ctx := service.EnsureRequestMetadata(c.Request.Context())
	service.SetImageRouteFamilyMetadata(ctx, service.PublicImageToolRouteFamily)
	service.SetImageActionMetadata(ctx, "generations")
	service.SetImageResolvedProviderMetadata(ctx, responsesImageToolResolvedProvider(platform))
	service.SetImageDisplayModelIDMetadata(ctx, requestedModel)
	service.SetImageTargetModelIDMetadata(ctx, toolModel)
	service.SetImageUpstreamEndpointMetadata(ctx, service.EndpointResponses)
	service.SetImageRequestFormatMetadata(ctx, service.EndpointResponses)
	service.SetImageRouteReasonMetadata(ctx, routeReason)
	service.SetImageRequestSurfaceMetadata(ctx, "responses_tool")
	c.Request = c.Request.WithContext(ctx)
}

func applyResponsesImageToolRuntimeMetadata(
	c *gin.Context,
	platform string,
	requestedModel string,
	toolModel string,
	protocolMode string,
	imageAction string,
	imageSizeTier string,
	imageCapabilityProfile string,
) {
	if c == nil || c.Request == nil {
		return
	}
	applyResponsesImageToolTraceMetadata(c, platform, requestedModel, toolModel, service.PublicImageToolRouteReason)
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	if action := strings.TrimSpace(strings.ToLower(imageAction)); action != "" {
		service.SetImageActionMetadata(ctx, action)
	}
	if mode := strings.TrimSpace(strings.ToLower(protocolMode)); mode != "" {
		service.SetImageProtocolModeMetadata(ctx, mode)
	}
	if tier := strings.TrimSpace(imageSizeTier); tier != "" {
		service.SetImageSizeTierMetadata(ctx, tier)
	}
	if profile := strings.TrimSpace(imageCapabilityProfile); profile != "" {
		service.SetImageCapabilityProfileMetadata(ctx, profile)
	}
	c.Request = c.Request.WithContext(ctx)
}
