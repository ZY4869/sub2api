package handler

import (
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/domain"
	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *GatewayHandler) beginGatewayMessagesRequest(c *gin.Context) (*gatewayMessagesRequest, bool) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return nil, false
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return nil, false
	}
	reqLog := requestLogger(c, "handler.gateway.messages", zap.Int64("user_id", subject.UserID), zap.Int64("api_key_id", apiKey.ID), zap.Any("group_id", apiKey.GroupID))
	return &gatewayMessagesRequest{apiKey: apiKey, subject: subject, reqLog: reqLog}, true
}

func (h *GatewayHandler) prepareGatewayMessagesRequest(c *gin.Context, req *gatewayMessagesRequest) bool {
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return false
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return false
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return false
	}
	req.body = body

	moderationInput := buildContentModerationRecordInput(c, service.ContentModerationSourceAnthropicMessages, service.PlatformAnthropic, "", body)
	if decision, err := checkContentModerationKeywordBlock(c.Request.Context(), h.contentModerationService, moderationInput); err != nil {
		req.reqLog.Warn("gateway.content_moderation_keyword_check_failed", zap.Error(err))
	} else if decision != nil {
		h.submitContentModerationFailedUsageRecordTask(
			"handler.gateway.messages",
			c,
			req.apiKey,
			strings.TrimSpace(gjson.GetBytes(body, "model").String()),
			gjson.GetBytes(body, "stream").Bool(),
			service.PlatformAnthropic,
			gatewayCompatiblePlatforms,
			decision,
		)
		contentModerationAnthropicBlockResponse(c, decision)
		return false
	}
	submitContentModerationAudit(c.Request.Context(), h.contentModerationService, moderationInput)
	setOpsRequestContext(c, "", false, body)

	parsedReq, err := service.ParseGatewayRequest(body, domain.PlatformAnthropic)
	if err != nil {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return false
	}
	req.parsedReq = parsedReq
	c.Request = c.Request.WithContext(service.EnsureRequestMetadata(c.Request.Context()))
	service.RecordClaudeCapabilityMetadata(c.Request.Context(), parsedReq.Capability)
	h.resolveParsedRequestModel(c.Request.Context(), parsedReq)

	req.publicRequestModel = parsedReq.Model
	if !h.resolveGatewayMessagesPublicCatalog(c, req) {
		return false
	}

	req.reqModel = parsedReq.Model
	req.reqStream = parsedReq.Stream
	req.reqLog = req.reqLog.With(zap.String("model", req.reqModel), zap.Bool("stream", req.reqStream))
	if isMaxTokensOneHaikuRequest(req.reqModel, parsedReq.MaxTokens, req.reqStream) {
		ctx := service.WithIsMaxTokensOneHaikuRequest(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
	}
	SetClaudeCodeClientContext(c, body, parsedReq)
	req.isClaudeCodeClient = service.IsClaudeCodeClient(c.Request.Context())
	if !h.checkClaudeCodeVersion(c) {
		return false
	}
	c.Request = c.Request.WithContext(service.WithThinkingEnabled(c.Request.Context(), parsedReq.ThinkingEnabled, h.metadataBridgeEnabled()))
	setOpsRequestContext(c, req.reqModel, req.reqStream, body)
	req.requestPayloadHash = service.HashUsageRequestPayload(body)
	if req.reqModel == "" {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "model is required")
		return false
	}

	req.selectionModel = h.gatewayService.ResolveAPIKeySelectionModel(c.Request.Context(), req.apiKey, "", req.publicRequestModel)
	if req.selectionModel == "" {
		h.publicCatalogUnavailableResponse(c, service.PublicCatalogResolutionNoMatch)
		return false
	}
	req.bindingSelectionModel = req.selectionModel
	if req.publicCatalogEntry != nil {
		req.bindingSelectionModel = req.publicRequestModel
	}
	if h.errorPassthroughService != nil {
		service.BindErrorPassthroughService(c, h.errorPassthroughService)
	}
	req.subscription, _ = middleware2.GetSubscriptionFromContext(c)
	if !h.acquireGatewayMessagesUserSlot(c, req) {
		return false
	}

	parsedReq.SessionContext = &service.SessionContext{ClientIP: ip.GetTrustedClientIP(c), UserAgent: c.GetHeader("User-Agent"), APIKeyID: req.apiKey.ID}
	req.selectedSessionHash = h.gatewayService.GenerateSessionHash(parsedReq)
	forcePlatform, hasForcePlatform := middleware2.GetForcePlatformFromContext(c)
	req.allowedPlatforms = gatewayCompatiblePlatforms
	if hasForcePlatform && strings.TrimSpace(forcePlatform) != "" {
		req.allowedPlatforms = []string{forcePlatform}
	}
	req.excludedGroupIDs = make(map[int64]struct{})
	return true
}

func (h *GatewayHandler) resolveGatewayMessagesPublicCatalog(c *gin.Context, req *gatewayMessagesRequest) bool {
	entry, status, err := h.gatewayService.ResolveAPIKeyPublishedPublicCatalogRuntimeStatus(c.Request.Context(), req.apiKey, "", req.publicRequestModel)
	if err != nil {
		req.reqLog.Warn("gateway.public_catalog_entry_resolve_failed", zap.Error(err))
		return true
	}
	if status == service.PublicCatalogResolutionNoMatch || status == service.PublicCatalogResolutionTimeWindowDenied {
		h.publicCatalogUnavailableResponse(c, status)
		return false
	}
	if status == service.PublicCatalogResolutionMatched {
		req.publicCatalogEntry = entry
		ctx := service.ApplyPublicCatalogEntryToParsedRequest(c.Request.Context(), req.parsedReq, entry)
		c.Request = c.Request.WithContext(ctx)
	}
	return true
}

func (h *GatewayHandler) acquireGatewayMessagesUserSlot(c *gin.Context, req *gatewayMessagesRequest) bool {
	maxWait := service.CalculateMaxWait(req.subject.Concurrency)
	canWait, err := h.concurrencyHelper.IncrementWaitCount(c.Request.Context(), req.subject.UserID, maxWait)
	if err != nil {
		req.reqLog.Warn("gateway.user_wait_counter_increment_failed", zap.Error(err))
	} else if !canWait {
		req.reqLog.Info("gateway.user_wait_queue_full", zap.Int("max_wait", maxWait))
		h.errorResponse(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later")
		return false
	}
	if err == nil && canWait {
		req.userWaitCounted = true
	}

	userReleaseFunc, err := h.concurrencyHelper.AcquireUserSlotWithWait(c, req.subject.UserID, req.subject.Concurrency, req.reqStream, &req.streamStarted)
	if err != nil {
		req.reqLog.Warn("gateway.user_slot_acquire_failed", zap.Error(err))
		h.handleConcurrencyError(c, err, "user", req.streamStarted)
		return false
	}
	if req.userWaitCounted {
		h.concurrencyHelper.DecrementWaitCount(c.Request.Context(), req.subject.UserID)
		req.userWaitCounted = false
	}
	req.userReleaseFunc = wrapReleaseOnDone(c.Request.Context(), userReleaseFunc)
	return true
}

func (h *GatewayHandler) releaseGatewayMessagesUserWait(c *gin.Context, req *gatewayMessagesRequest) {
	if req != nil && req.userWaitCounted {
		h.concurrencyHelper.DecrementWaitCount(c.Request.Context(), req.subject.UserID)
		req.userWaitCounted = false
	}
}
