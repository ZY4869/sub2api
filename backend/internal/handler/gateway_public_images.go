package handler

import (
	"bytes"
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) ResolvePublicImageRoute(c *gin.Context, inboundEndpoint string) (service.PublicImageRouteDecision, bool) {
	if h == nil || h.gatewayService == nil || c == nil || c.Request == nil {
		return service.PublicImageRouteDecision{}, false
	}

	apiKey, ok := middleware.GetAPIKeyFromContext(c)
	if !ok || apiKey == nil {
		c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "authentication_error",
				"message": "Invalid API key",
			},
		})
		return service.PublicImageRouteDecision{}, false
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": "Failed to read request body",
			},
		})
		return service.PublicImageRouteDecision{}, false
	}
	c.Request.Body = ioNopCloserBytes(body)

	rawContentType := strings.TrimSpace(c.GetHeader("Content-Type"))
	requestedModel, requestFormat, detectErr := detectPublicImageRouteInput(body, rawContentType)
	if detectErr != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "invalid_request_error",
				"message": detectErr.Error(),
			},
		})
		return service.PublicImageRouteDecision{}, false
	}

	decision, resolveErr := h.gatewayService.ResolvePublicImageRoute(c.Request.Context(), apiKey, inboundEndpoint, requestedModel)
	if resolveErr != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    "api_error",
				"message": "Failed to resolve public image route",
			},
		})
		return service.PublicImageRouteDecision{}, false
	}
	if !decision.Supported {
		statusCode := decision.StatusCode
		if statusCode <= 0 {
			statusCode = http.StatusBadRequest
		}
		errType := strings.TrimSpace(decision.ErrorType)
		if errType == "" {
			errType = "invalid_request_error"
		}
		errorCode := strings.TrimSpace(decision.ErrorCode)
		if errorCode == "" {
			errorCode = service.GatewayReasonPublicEndpointUnsupported
		}
		errorMessage := strings.TrimSpace(decision.ErrorMessage)
		if errorMessage == "" {
			errorMessage = "Unable to resolve image route"
		}
		c.AbortWithStatusJSON(statusCode, gin.H{
			"type": "error",
			"error": gin.H{
				"type":    errType,
				"message": errorMessage,
				"code":    errorCode,
				"reason":  errorCode,
			},
		})
		return decision, false
	}

	decision.RequestFormat = firstNonEmptyHandlerString(strings.TrimSpace(requestFormat), decision.RequestFormat)
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	service.SetImageRouteFamilyMetadata(ctx, decision.ImageRouteFamily)
	service.SetImageActionMetadata(ctx, decision.ImageAction)
	service.SetImageResolvedProviderMetadata(ctx, decision.ResolvedProvider)
	service.SetImageDisplayModelIDMetadata(ctx, decision.DisplayModelID)
	service.SetImageTargetModelIDMetadata(ctx, decision.TargetModelID)
	service.SetImageUpstreamEndpointMetadata(ctx, decision.UpstreamEndpoint)
	service.SetImageRequestFormatMetadata(ctx, decision.RequestFormat)
	service.SetImageRouteReasonMetadata(ctx, decision.RouteReason)
	c.Request = c.Request.WithContext(ctx)

	return decision, true
}

func (h *GatewayHandler) GeminiOpenAICompatImagesGeneration(c *gin.Context) {
	attachGeminiPublicProtocolContext(c)
	h.forwardGeminiPassthrough(c, service.GeminiPublicPassthroughInput{
		UpstreamPath: "/v1beta/openai/images/generations",
	})
}

func detectPublicImageRouteInput(body []byte, contentType string) (string, string, error) {
	trimmedType := strings.TrimSpace(strings.ToLower(contentType))
	requestFormat := "application/json"
	if strings.HasPrefix(trimmedType, "multipart/form-data") {
		requestFormat = "multipart/form-data"
	}
	if len(body) == 0 {
		return "", requestFormat, nil
	}
	model, err := service.DetectOpenAIImageRequestModel(body, contentType)
	if err != nil {
		if requestFormat == "multipart/form-data" {
			return "", requestFormat, err
		}
		return "", requestFormat, err
	}
	return strings.TrimSpace(model), requestFormat, nil
}

func ioNopCloserBytes(body []byte) *readCloserBytes {
	return &readCloserBytes{Reader: bytes.NewReader(body)}
}

type readCloserBytes struct {
	*bytes.Reader
}

func (r *readCloserBytes) Close() error { return nil }
