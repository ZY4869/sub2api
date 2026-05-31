package handler

import (
	"net/http"
	"strings"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) prepareEmbeddingsRequest(c *gin.Context) (*openAIEmbeddingsRequest, bool) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return nil, false
	}
	if apiKey.Group != nil {
		applyOpenAIPlatformContext(c, apiKey.Group.Platform)
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return nil, false
	}
	reqLog := requestLogger(c, "handler.openai_gateway.embeddings",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return nil, false
	}

	body, reqModel, ok := h.readAndValidateEmbeddingsBody(c)
	if !ok {
		return nil, false
	}
	reqLog = reqLog.With(zap.String("model", reqModel))
	setOpsRequestContext(c, reqModel, false, body)
	prepared := &openAIEmbeddingsRequest{
		apiKey:              apiKey,
		subject:             subject,
		body:                body,
		reqModel:            reqModel,
		publicRequestModel:  reqModel,
		runtimeRequestModel: reqModel,
		requestPayloadHash:  service.HashUsageRequestPayload(body),
		reqLog:              reqLog,
	}
	if !h.resolveEmbeddingsPublicCatalogEntry(c, prepared) {
		return nil, false
	}
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	return prepared, true
}

func (h *OpenAIGatewayHandler) readAndValidateEmbeddingsBody(c *gin.Context) ([]byte, string, bool) {
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return nil, "", false
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return nil, "", false
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return nil, "", false
	}
	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return nil, "", false
	}
	modelResult := gjson.GetBytes(body, "model")
	if !modelResult.Exists() || modelResult.Type != gjson.String || strings.TrimSpace(modelResult.String()) == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return nil, "", false
	}
	return body, strings.TrimSpace(modelResult.String()), true
}

func (h *OpenAIGatewayHandler) resolveEmbeddingsPublicCatalogEntry(c *gin.Context, req *openAIEmbeddingsRequest) bool {
	entry, matched, active, err := h.gatewayService.ResolveAPIKeyPublishedPublicCatalogRuntime(
		c.Request.Context(),
		req.apiKey,
		service.OpenAIPlatformFromContext(c.Request.Context()),
		req.reqModel,
	)
	if err != nil {
		req.reqLog.Warn("openai_embeddings.public_catalog_entry_resolve_failed", zap.Error(err))
		return true
	}
	if active && !matched {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", service.PublicCatalogModelUnavailableMessage)
		return false
	}
	if matched {
		req.publicCatalogEntry = entry
		req.runtimeRequestModel = service.NormalizeModelCatalogModelID(firstNonEmptyHandlerString(entry.SourceModelID, req.reqModel))
		c.Request = c.Request.WithContext(service.AttachPublishedPublicCatalogEntry(c.Request.Context(), entry))
	}
	return true
}
