package handler

import (
	"io"
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) GeminiV1BetaFiles(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error) {
		return h.geminiNativeService.ForwardGoogleFiles(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GeminiV1BetaFileUpload(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error) {
		return h.geminiNativeService.ForwardGoogleFiles(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GeminiV1BetaFileDownload(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error) {
		return h.geminiNativeService.ForwardGoogleFileDownload(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GeminiV1BetaBatches(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error) {
		return h.geminiNativeService.ForwardGoogleBatches(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) VertexBatchPredictionJobs(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error) {
		return h.geminiNativeService.ForwardVertexBatchPredictionJobs(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GoogleBatchArchiveBatch(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error) {
		return h.geminiNativeService.ForwardGoogleArchiveBatch(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GoogleBatchArchiveFileDownload(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error) {
		return h.geminiNativeService.ForwardGoogleArchiveFileDownload(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) forwardGoogleBatch(c *gin.Context, forwarder func(service.GoogleBatchForwardInput) (service.GoogleBatchUpstreamResult, *service.Account, error)) {
	if h == nil || h.geminiNativeService == nil {
		googleErrorKey(c, http.StatusServiceUnavailable, "gateway.gemini.batch_service_missing", "Gemini batch service not configured")
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
	currentAPIKey, _, err := resolveSelectedGatewayAPIKey(
		c,
		h.settingService,
		h.gatewayService,
		h.billingCacheService,
		apiKey,
		subscription,
		"",
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
	body, openBody, cleanupBody, contentLength, err := readGoogleBatchForwardBody(c)
	if err != nil {
		return
	}
	if cleanupBody != nil {
		defer cleanupBody()
	}
	result, _, err := forwarder(service.GoogleBatchForwardInput{
		GroupID:        currentAPIKey.GroupID,
		APIKeyID:       currentAPIKey.ID,
		APIKey:         currentAPIKey,
		UserID:         authSubject.UserID,
		BillingType:    resolveGoogleBatchBillingType(subscription),
		SubscriptionID: resolveGoogleBatchSubscriptionID(subscription),
		Method:         c.Request.Method,
		Path:           c.Request.URL.Path,
		RawQuery:       c.Request.URL.RawQuery,
		Headers:        c.Request.Header.Clone(),
		Body:           body,
		OpenBody:       openBody,
		ContentLength:  contentLength,
	})
	if err != nil {
		googleErrorFromServiceError(c, err)
		return
	}
	writeGoogleBatchUpstreamResponse(c, result)
}

func resolveGoogleBatchBillingType(subscription *service.UserSubscription) int8 {
	if subscription != nil && subscription.ID > 0 {
		return service.BillingTypeSubscription
	}
	return service.BillingTypeBalance
}

func resolveGoogleBatchSubscriptionID(subscription *service.UserSubscription) *int64 {
	if subscription == nil || subscription.ID <= 0 {
		return nil
	}
	value := subscription.ID
	return &value
}

func readGoogleBatchForwardBody(c *gin.Context) ([]byte, func() (io.ReadCloser, error), func(), int64, error) {
	if c == nil || c.Request == nil {
		return nil, nil, nil, 0, nil
	}
	switch strings.ToUpper(strings.TrimSpace(c.Request.Method)) {
	case http.MethodGet, http.MethodHead:
		return nil, nil, nil, 0, nil
	}
	if shouldStreamGoogleBatchRequestBody(c.Request.Method, c.Request.URL.Path) {
		if c.Request.Body == nil {
			return nil, nil, nil, c.Request.ContentLength, nil
		}
		replayableBody, err := newReplayableGoogleBatchBody(c.Request.Body)
		if err != nil {
			googleErrorKey(c, http.StatusInternalServerError, "gateway.gemini.prepare_body_failed", "Failed to prepare request body")
			return nil, nil, nil, 0, err
		}
		return nil, replayableBody.Open, replayableBody.Cleanup, c.Request.ContentLength, nil
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			googleErrorBodyTooLarge(c, maxErr.Limit)
			return nil, nil, nil, 0, err
		}
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.read_body_failed", "Failed to read request body")
		return nil, nil, nil, 0, err
	}
	return body, nil, nil, int64(len(body)), nil
}

func shouldStreamGoogleBatchRequestBody(method string, path string) bool {
	if !strings.EqualFold(strings.TrimSpace(method), http.MethodPost) {
		return false
	}
	trimmed := strings.ToLower(strings.TrimSpace(path))
	return trimmed == "/upload/v1beta/files" || strings.HasPrefix(trimmed, "/upload/v1beta/filesearchstores/")
}

func attachGeminiPublicProtocolContext(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	ctx = service.WithGeminiPublicProtocol(ctx, geminiInboundPublicProtocol(c))
	c.Request = c.Request.WithContext(ctx)
	applyGeminiPublicPathMetadata(c, "")
}

func geminiInboundPublicProtocol(c *gin.Context) string {
	switch GetInboundEndpoint(c) {
	case EndpointVertexSyncModels, EndpointVertexBatchJobs:
		return service.UpstreamProviderVertexAI
	default:
		return service.UpstreamProviderAIStudio
	}
}
