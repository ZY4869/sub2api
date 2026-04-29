package handler

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *GatewayHandler) forwardGeminiPassthrough(c *gin.Context, input service.GeminiPublicPassthroughInput) {
	requestStart := time.Now()
	applyGeminiPublicPathMetadata(c, input.UpstreamPath)
	passthroughService := h.resolveGeminiPassthroughService(c, input)
	if passthroughService == nil {
		googleErrorKey(c, http.StatusServiceUnavailable, "gateway.gemini.passthrough_service_missing", "Gemini passthrough service not configured")
		return
	}

	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		googleErrorKey(c, http.StatusUnauthorized, "gateway.gemini.invalid_api_key", "Invalid API key")
		return
	}
	authSubject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		googleErrorKey(c, http.StatusInternalServerError, "gateway.gemini.user_context_missing", "User context not found")
		return
	}
	subscription, _ := middleware.GetSubscriptionFromContext(c)
	body, openBody, cleanupBody, contentLength, err := readGoogleBatchForwardBody(c)
	if err != nil {
		return
	}
	if cleanupBody != nil {
		defer cleanupBody()
	}
	if strings.TrimSpace(input.RequestedModel) == "" {
		input.RequestedModel = detectGeminiPassthroughRequestedModel(c.Request.URL.Path, body)
	}
	stream := geminiPassthroughStreamRequested(c, body)
	setOpsRequestContext(c, input.RequestedModel, stream, body)

	currentAPIKey, currentSubscription, err := resolveSelectedGatewayAPIKey(
		c,
		h.settingService,
		h.gatewayService,
		h.billingCacheService,
		apiKey,
		subscription,
		input.RequestedModel,
		[]string{service.PlatformGemini},
		nil,
	)
	if err != nil {
		googleErrorFromServiceError(c, err)
		return
	}
	if !middleware.HasForcePlatform(c) && currentAPIKey.Group != nil && strings.TrimSpace(currentAPIKey.Group.Platform) != service.PlatformGemini {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.group_platform_invalid", "API key group platform is not gemini")
		return
	}

	reqLog := requestLogger(
		c,
		"handler.gemini_v1beta.passthrough",
		zap.Int64("user_id", authSubject.UserID),
		zap.Int64("api_key_id", currentAPIKey.ID),
		zap.Any("group_id", currentAPIKey.GroupID),
		zap.String("path", strings.TrimSpace(c.Request.URL.Path)),
	)

	imageSizeTier := service.ResolveOpenAIImageSizeTier(service.DetectOpenAIImageRequestSize(body, c.GetHeader("Content-Type")))
	expectedImageCount := service.DetectOpenAIImageRequestN(body, c.GetHeader("Content-Type"))
	reservedImageUnits := 0
	imageCountSettled := false
	if currentAPIKey.EffectiveImageCountBillingEnabled() && GetInboundEndpoint(c) == service.EndpointImagesGen {
		if h.apiKeyService == nil {
			reqLog.Error("api_key_service_missing_for_image_quota")
			googleErrorKey(c, http.StatusInternalServerError, "api_key.image_count_quota_unavailable", "Image quota service unavailable")
			return
		}
		reservedImageUnits = currentAPIKey.ImageCountUnitsForTier(expectedImageCount, imageSizeTier)
		ok, reserveErr := h.apiKeyService.TryReserveImageCount(c.Request.Context(), currentAPIKey.ID, reservedImageUnits)
		if reserveErr != nil {
			reqLog.Error("api_key_image_count_reserve_failed", zap.Error(reserveErr), zap.String("image_size_tier", imageSizeTier), zap.Int("image_count", expectedImageCount), zap.Int("reserved_units", reservedImageUnits))
			googleErrorKey(c, http.StatusInternalServerError, "api_key.image_count_reserve_failed", "Failed to reserve image quota")
			return
		}
		if !ok {
			googleErrorWithReason(c, http.StatusTooManyRequests, "IMAGE_ONLY_KEY_IMAGE_QUOTA_EXHAUSTED", "api_key.image_count_quota_exhausted", "图片数量额度已用完")
			return
		}
		reqLog.Info("api_key_image_count_reserved", zap.String("image_size_tier", imageSizeTier), zap.Int("image_count", expectedImageCount), zap.Int("reserved_units", reservedImageUnits), zap.Int("max", currentAPIKey.ImageMaxCount))
		defer func() {
			if reservedImageUnits <= 0 || imageCountSettled {
				return
			}
			if err := h.apiKeyService.RollbackImageCount(c.Request.Context(), currentAPIKey.ID, reservedImageUnits); err != nil {
				reqLog.Error("api_key_image_count_rollback_failed", zap.Error(err), zap.String("image_size_tier", imageSizeTier), zap.Int("rollback_units", reservedImageUnits))
				return
			}
			reqLog.Info("api_key_image_count_rolled_back", zap.String("image_size_tier", imageSizeTier), zap.Int("rollback_units", reservedImageUnits))
		}()
	}

	result, err := passthroughService.ForwardGeminiPassthrough(c.Request.Context(), service.GeminiPublicPassthroughInput{
		GoogleBatchForwardInput: service.GoogleBatchForwardInput{
			GroupID:        currentAPIKey.GroupID,
			APIKeyID:       currentAPIKey.ID,
			APIKey:         currentAPIKey,
			UserID:         authSubject.UserID,
			BillingType:    resolveGoogleBatchBillingType(currentSubscription),
			SubscriptionID: resolveGoogleBatchSubscriptionID(currentSubscription),
			Method:         c.Request.Method,
			Path:           c.Request.URL.Path,
			RawQuery:       c.Request.URL.RawQuery,
			Headers:        c.Request.Header.Clone(),
			Body:           body,
			OpenBody:       openBody,
			ContentLength:  contentLength,
		},
		RequestedModel: input.RequestedModel,
		ResourceKind:   input.ResourceKind,
		UpstreamPath:   input.UpstreamPath,
	})
	if err != nil {
		reqLog.Warn("gemini.passthrough_failed", zap.Error(err))
		googleErrorFromServiceError(c, err)
		if result != nil && result.Account != nil {
			var failoverErr *service.UpstreamFailoverError
			if upstreamResult, ok := result.Response.(*service.UpstreamHTTPResult); ok && upstreamResult != nil {
				failoverErr = &service.UpstreamFailoverError{
					StatusCode:      upstreamResult.StatusCode,
					ResponseHeaders: upstreamResult.Headers,
					ResponseBody:    upstreamResult.Body,
				}
			}
			h.submitFailedUsageRecordTask(
				"handler.gemini_v1beta.passthrough",
				c,
				currentAPIKey,
				currentSubscription,
				result.Account,
				input.RequestedModel,
				stream,
				time.Since(requestStart),
				service.PlatformGemini,
				failoverErr,
				err,
			)
		}
		return
	}
	if result == nil || result.Account == nil {
		googleErrorKey(c, http.StatusBadGateway, "gateway.gemini.upstream_empty", "Empty upstream response")
		return
	}

	if reservedImageUnits > 0 && !imageCountSettled {
		actualCount := expectedImageCount
		actualTier := imageSizeTier
		if upstreamResult, ok := result.Response.(*service.UpstreamHTTPResult); ok && upstreamResult != nil {
			if count := service.CountOpenAIImageResponse(upstreamResult.Body); count > 0 {
				actualCount = count
			}
		}
		if result.ForwardResult != nil && strings.TrimSpace(result.ForwardResult.ImageSize) != "" {
			actualTier = service.ResolveOpenAIImageSizeTier(result.ForwardResult.ImageSize)
		}
		imageCountSettled = settleAPIKeyImageCountUnits(c.Request.Context(), reqLog, h.apiKeyService, currentAPIKey, reservedImageUnits, actualCount, actualTier)
	}

	requestType := service.RequestTypeSync
	if result.ForwardResult != nil && result.ForwardResult.Stream {
		requestType = service.RequestTypeStream
	}
	upstreamModel := input.RequestedModel
	if result.ForwardResult != nil {
		upstreamModel = firstNonEmptyHandlerString(result.ForwardResult.UpstreamModel, upstreamModel)
	}
	setOpsSelectedAccountDetails(c, result.Account)
	setOpsEndpointContext(c, upstreamModel, requestType)

	writeGoogleBatchUpstreamResponse(c, result.Response)

	if result.ForwardResult == nil || currentAPIKey.User == nil {
		return
	}
	requestPayloadHash := service.HashUsageRequestPayload(body)
	inboundEndpoint := GetInboundEndpoint(c)
	rawInboundPath := strings.TrimSpace(c.Request.URL.Path)
	upstreamEndpoint := strings.TrimSpace(c.Request.URL.Path)
	usageDecision := service.DecideGeminiSuccessUsagePersistence(inboundEndpoint, rawInboundPath, body)
	if !usageDecision.Persist {
		reqLog.Info("gemini.usage_record_skipped", zap.String("reason", usageDecision.Reason), zap.String("operation_type", usageDecision.OperationType), zap.String("inbound_endpoint", inboundEndpoint))
		return
	}
	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetClientIP(c)
	h.submitUsageRecordTask(func(ctx context.Context) {
		if err := h.gatewayService.RecordUsageWithLongContext(ctx, &service.RecordUsageLongContextInput{
			Result:                result.ForwardResult,
			APIKey:                currentAPIKey,
			User:                  currentAPIKey.User,
			Account:               result.Account,
			Subscription:          currentSubscription,
			InboundEndpoint:       inboundEndpoint,
			RawInboundPath:        rawInboundPath,
			UpstreamEndpoint:      upstreamEndpoint,
			UserAgent:             userAgent,
			IPAddress:             clientIP,
			RequestBody:           body,
			RequestPayloadHash:    requestPayloadHash,
			LongContextThreshold:  200000,
			LongContextMultiplier: 2.0,
			APIKeyService:         h.apiKeyService,
		}); err != nil {
			reqLog.Error("gemini.passthrough_record_usage_failed", zap.Error(err))
		}
	})
}

type geminiPassthroughForwarder interface {
	ForwardGeminiPassthrough(ctx context.Context, input service.GeminiPublicPassthroughInput) (*service.GeminiPublicPassthroughOutput, error)
}

func (h *GatewayHandler) resolveGeminiPassthroughService(c *gin.Context, input service.GeminiPublicPassthroughInput) geminiPassthroughForwarder {
	if h == nil {
		return nil
	}
	path := ""
	if c != nil && c.Request != nil && c.Request.URL != nil {
		path = strings.ToLower(strings.TrimSpace(c.Request.URL.Path))
	}
	switch {
	case input.ResourceKind == service.UpstreamResourceKindGeminiInteraction:
		return h.geminiInteractionsService
	case strings.Contains(path, "/v1alpha/authtokens"),
		strings.Contains(path, "/v1beta/live"),
		strings.EqualFold(strings.TrimSpace(input.UpstreamPath), service.GeminiLiveAuthTokensPath):
		return h.geminiLiveService
	case strings.Contains(path, "/v1beta/openai/"):
		return h.geminiCompatService
	default:
		return h.geminiNativeService
	}
}

func detectGeminiPassthroughRequestedModel(path string, body []byte) string {
	if model := strings.TrimSpace(gjson.GetBytes(body, "model").String()); model != "" {
		return model
	}
	trimmed := strings.TrimSpace(path)
	lower := strings.ToLower(trimmed)
	for _, marker := range []string{"/models/", "/tunedmodels/", "/dynamic/"} {
		if idx := strings.Index(lower, marker); idx >= 0 {
			modelPart := trimmed[idx+len(marker):]
			for _, sep := range []string{":", "?", "/"} {
				if cut := strings.Index(modelPart, sep); cut >= 0 {
					modelPart = modelPart[:cut]
				}
			}
			return strings.TrimSpace(modelPart)
		}
	}
	return ""
}

func geminiPassthroughStreamRequested(c *gin.Context, body []byte) bool {
	if gjson.GetBytes(body, "stream").Bool() {
		return true
	}
	if c == nil || c.Request == nil {
		return false
	}
	return strings.Contains(strings.ToLower(strings.TrimSpace(c.Request.URL.RawQuery)), "stream=true")
}

func firstNonEmptyHandlerString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}
