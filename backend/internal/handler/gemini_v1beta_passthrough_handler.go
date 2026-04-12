package handler

import (
	"context"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *GatewayHandler) forwardGeminiPassthrough(c *gin.Context, input service.GeminiPublicPassthroughInput) {
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
		return
	}
	if result == nil || result.Account == nil {
		googleErrorKey(c, http.StatusBadGateway, "gateway.gemini.upstream_empty", "Empty upstream response")
		return
	}

	requestType := service.RequestTypeSync
	if result.ForwardResult != nil && result.ForwardResult.Stream {
		requestType = service.RequestTypeStream
	}
	upstreamModel := input.RequestedModel
	if result.ForwardResult != nil {
		upstreamModel = firstNonEmptyHandlerString(result.ForwardResult.UpstreamModel, upstreamModel)
	}
	setOpsSelectedAccount(c, result.Account.ID, result.Account.Platform)
	setOpsEndpointContext(c, upstreamModel, requestType)

	writeGoogleBatchUpstreamResponse(c, result.Response)

	if result.ForwardResult == nil || currentAPIKey.User == nil {
		return
	}
	requestPayloadHash := service.HashUsageRequestPayload(body)
	inboundEndpoint := strings.TrimSpace(c.Request.URL.Path)
	upstreamEndpoint := strings.TrimSpace(c.Request.URL.Path)
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
	case strings.Contains(path, "/v1beta/live"):
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
	if idx := strings.Index(trimmed, "/models/"); idx >= 0 {
		modelPart := trimmed[idx+len("/models/"):]
		for _, sep := range []string{":", "?", "/"} {
			if cut := strings.Index(modelPart, sep); cut >= 0 {
				modelPart = modelPart[:cut]
			}
		}
		return strings.TrimSpace(modelPart)
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
