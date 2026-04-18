package handler

import (
	"bytes"
	"io"
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

func (h *GatewayHandler) VertexModels(c *gin.Context) {
	attachVertexGeminiSurface(c, "vertex_strict")
	h.GeminiV1BetaModels(c)
}

func (h *GatewayHandler) VertexModelsSimplified(c *gin.Context) {
	if h == nil || h.gatewayService == nil || h.geminiNativeService == nil {
		googleErrorKey(c, http.StatusServiceUnavailable, "gateway.gemini.batch_service_missing", "Gemini batch service not configured")
		return
	}
	attachGeminiPublicProtocolContext(c)
	attachVertexGeminiSurface(c, "vertex_simplified")
	if c == nil || c.Request == nil {
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.request_failed", "Request failed")
		return
	}

	modelName, action, err := parseGeminiModelAction(strings.TrimPrefix(c.Param("modelAction"), "/"))
	if err != nil {
		messageKey, fallback, mismatchKind := geminiModelActionRouteMismatchDetails(err)
		writeGeminiRouteMismatch(c, mismatchKind, messageKey, fallback)
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			googleErrorBodyTooLarge(c, maxErr.Limit)
			return
		}
		googleErrorKey(c, http.StatusBadRequest, "gateway.gemini.read_body_failed", "Failed to read request body")
		return
	}

	normalizedBody, err := service.NormalizeSimplifiedVertexModelRequest(modelName, action, body)
	if err != nil {
		googleErrorFromServiceError(c, err)
		return
	}

	ctx := service.WithGeminiPublicProtocolStrict(service.EnsureRequestMetadata(c.Request.Context()), service.UpstreamProviderVertexAI)
	service.SetGeminiSurfaceMetadata(ctx, "vertex_simplified")
	c.Request = c.Request.WithContext(ctx)
	c.Request.Body = io.NopCloser(bytes.NewReader(normalizedBody))
	c.Request.ContentLength = int64(len(normalizedBody))

	h.GeminiV1BetaModels(c)
}

func writeGeminiRouteMismatch(c *gin.Context, mismatchKind string, messageKey string, fallback string) {
	service.SetOpsUpstreamError(c, 0, "", "")
	googleErrorWithReason(c, http.StatusNotFound, service.GatewayReasonRouteMismatch, messageKey, fallback)
	_ = mismatchKind
}

func attachVertexGeminiSurface(c *gin.Context, surface string) {
	if c == nil || c.Request == nil {
		return
	}
	ctx := service.EnsureRequestMetadata(c.Request.Context())
	service.SetGeminiSurfaceMetadata(ctx, surface)
	c.Request = c.Request.WithContext(ctx)
}
