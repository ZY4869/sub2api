package handler

import (
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) GeminiV1BetaFiles(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error) {
		return h.geminiCompatService.ForwardGoogleFiles(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GeminiV1BetaFileUpload(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error) {
		return h.geminiCompatService.ForwardGoogleFiles(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GeminiV1BetaFileDownload(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error) {
		return h.geminiCompatService.ForwardGoogleFileDownload(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GeminiV1BetaBatches(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error) {
		return h.geminiCompatService.ForwardGoogleBatches(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) VertexBatchPredictionJobs(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error) {
		return h.geminiCompatService.ForwardVertexBatchPredictionJobs(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GoogleBatchArchiveBatch(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error) {
		return h.geminiCompatService.ForwardGoogleArchiveBatch(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) GoogleBatchArchiveFileDownload(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGoogleBatch(c, func(input service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error) {
		return h.geminiCompatService.ForwardGoogleArchiveFileDownload(c.Request.Context(), input)
	})
}

func (h *GatewayHandler) forwardGoogleBatch(c *gin.Context, forwarder func(service.GoogleBatchForwardInput) (*service.UpstreamHTTPResult, *service.Account, error)) {
	if h == nil || h.geminiCompatService == nil {
		googleError(c, http.StatusServiceUnavailable, "Gemini batch service not configured")
		return
	}
	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		googleError(c, http.StatusUnauthorized, "Invalid API key")
		return
	}
	authSubject, ok := middleware.GetAuthSubjectFromContext(c)
	if !ok {
		googleError(c, http.StatusInternalServerError, "User context not found")
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
		googleError(c, http.StatusBadRequest, "API key group platform is not gemini")
		return
	}
	body, err := readOptionalGoogleBatchBody(c)
	if err != nil {
		return
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
	})
	if err != nil {
		googleErrorFromServiceError(c, err)
		return
	}
	writeUpstreamResponse(c, result)
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

func readOptionalGoogleBatchBody(c *gin.Context) ([]byte, error) {
	if c == nil || c.Request == nil {
		return nil, nil
	}
	switch strings.ToUpper(strings.TrimSpace(c.Request.Method)) {
	case http.MethodGet, http.MethodHead:
		return nil, nil
	}
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			googleError(c, http.StatusRequestEntityTooLarge, buildBodyTooLargeMessage(maxErr.Limit))
			return nil, err
		}
		googleError(c, http.StatusBadRequest, "Failed to read request body")
		return nil, err
	}
	return body, nil
}

func attachGeminiPublicProtocolContext(c *gin.Context) {
	if c == nil || c.Request == nil {
		return
	}
	c.Request = c.Request.WithContext(service.WithGeminiPublicProtocol(c.Request.Context(), geminiInboundPublicProtocol(c)))
}

func geminiInboundPublicProtocol(c *gin.Context) string {
	switch GetInboundEndpoint(c) {
	case EndpointVertexSyncModels, EndpointVertexBatchJobs:
		return service.UpstreamProviderVertexAI
	default:
		return service.UpstreamProviderAIStudio
	}
}
