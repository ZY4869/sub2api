package handler

import (
	"context"
	"errors"
	"net/http"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
	"github.com/Wei-Shaw/sub2api/internal/pkg/ip"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/tidwall/gjson"
	"go.uber.org/zap"
)

func (h *OpenAIGatewayHandler) AlphaSearch(c *gin.Context) {
	setOpenAIClientTransportHTTP(c)
	requestStart := time.Now()
	streamStarted := false

	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	if apiKey.Group != nil {
		applyOpenAIPlatformContext(c, apiKey.Group.Platform)
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "User context not found")
		return
	}
	reqLog := requestLogger(
		c,
		"handler.openai_gateway.alpha_search",
		zap.Int64("user_id", subject.UserID),
		zap.Int64("api_key_id", apiKey.ID),
		zap.Any("group_id", apiKey.GroupID),
	)
	if !h.ensureResponsesDependencies(c, reqLog) {
		return
	}

	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		if maxErr, ok := extractMaxBytesError(err); ok {
			h.errorResponse(c, http.StatusRequestEntityTooLarge, "invalid_request_error", buildBodyTooLargeMessage(maxErr.Limit))
			return
		}
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to read request body")
		return
	}
	if len(body) == 0 {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Request body is empty")
		return
	}
	if !gjson.ValidBytes(body) {
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", "Failed to parse request body")
		return
	}
	setOpsRequestContext(c, service.OpenAIAlphaSearchUsageModel, false, body)
	requestPayloadHash := service.HashUsageRequestPayload(body)

	subscription, _ := middleware2.GetSubscriptionFromContext(c)
	service.SetOpsLatencyMs(c, service.OpsAuthLatencyMsKey, time.Since(requestStart).Milliseconds())
	userReleaseFunc, acquired := h.acquireResponsesUserSlot(c, subject.UserID, apiKey.ID, subject.Concurrency, false, &streamStarted, reqLog)
	if !acquired {
		return
	}
	if userReleaseFunc != nil {
		defer userReleaseFunc()
	}

	excludedGroupIDs := make(map[int64]struct{})
	currentAPIKey, currentSubscription, err := resolveSelectedOpenAIAPIKey(
		c,
		h.settingService,
		h.gatewayService,
		h.billingCacheService,
		apiKey,
		subscription,
		service.OpenAIAlphaSearchUsageModel,
		openAICompatiblePlatforms,
		excludedGroupIDs,
	)
	if err != nil {
		reqLog.Info("openai.alpha_search.group_selection_failed", zap.Error(err))
		status, code, message := groupSelectionErrorDetails(err)
		h.errorResponse(c, status, code, message)
		return
	}
	if currentAPIKey.Group == nil || currentAPIKey.Group.Platform != service.PlatformOpenAI {
		h.errorResponse(c, http.StatusNotFound, "not_found_error", "The requested endpoint is not supported for this platform")
		return
	}
	applyOpenAIPlatformContext(c, currentAPIKey.Group.Platform)

	sessionHash, _, _ := h.gatewayService.ResolveSessionHashWithSource(c, body, "")
	routingStart := time.Now()
	selection, scheduleDecision, err := h.gatewayService.SelectAccountWithSchedulerForCapability(
		c.Request.Context(),
		currentAPIKey.GroupID,
		"",
		sessionHash,
		"",
		nil,
		service.OpenAIUpstreamTransportAny,
		service.OpenAIEndpointCapabilityAlphaSearch,
	)
	if err != nil {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
		reqLog.Warn("openai.alpha_search.account_select_failed", zap.Error(err))
		if errors.Is(err, service.ErrOpenAIModelNotFound) {
			h.handleOpenAIModelNotFound(c, service.OpenAIAlphaSearchUsageModel, streamStarted)
			return
		}
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "Service temporarily unavailable")
		return
	}
	if selection == nil || selection.Account == nil {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
		h.errorResponse(c, http.StatusServiceUnavailable, "api_error", "No available accounts")
		return
	}
	account := selection.Account
	setOpsSelectedAccountDetails(c, account)
	setOpsEndpointContext(c, service.OpenAIAlphaSearchUsageModel, service.RequestTypeSync)
	reqLog.Debug("openai.alpha_search.account_schedule_decision",
		zap.String("layer", scheduleDecision.Layer),
		zap.Int("candidate_count", scheduleDecision.CandidateCount),
		zap.Int64("latency_ms", scheduleDecision.LatencyMs),
	)
	service.SetOpsLatencyMs(c, service.OpsRoutingLatencyMsKey, time.Since(routingStart).Milliseconds())

	accountReleaseFunc, acquired := h.acquireResponsesAccountSlot(c, currentAPIKey.GroupID, sessionHash, selection, false, &streamStarted, reqLog)
	if !acquired {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
		return
	}
	forwardStart := time.Now()
	result, err := h.gatewayService.ForwardAlphaSearch(c.Request.Context(), c, account, body)
	forwardDurationMs := time.Since(forwardStart).Milliseconds()
	if accountReleaseFunc != nil {
		accountReleaseFunc()
	}
	upstreamLatencyMs, _ := getContextInt64(c, service.OpsUpstreamLatencyMsKey)
	responseLatencyMs := forwardDurationMs
	if upstreamLatencyMs > 0 && forwardDurationMs > upstreamLatencyMs {
		responseLatencyMs = forwardDurationMs - upstreamLatencyMs
	}
	service.SetOpsLatencyMs(c, service.OpsResponseLatencyMsKey, responseLatencyMs)
	if err != nil {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
		h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, false, nil)
		wroteFallback := h.ensureForwardErrorResponse(c, false)
		reqLog.Warn("openai.alpha_search.forward_failed", zap.Int64("account_id", account.ID), zap.Bool("fallback_error_response_written", wroteFallback), zap.Error(err))
		return
	}
	h.gatewayService.ReportOpenAIAccountScheduleResult(account.ID, result != nil && result.StatusCode >= 200 && result.StatusCode < 300, nil)
	if result == nil || result.StatusCode < 200 || result.StatusCode >= 300 {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, currentAPIKey)
		reqLog.Debug("openai.alpha_search.not_billed", zap.Int64("account_id", account.ID), zap.Int("status_code", alphaSearchStatusCode(result)))
		return
	}

	userAgent := c.GetHeader("User-Agent")
	clientIP := ip.GetTrustedClientIP(c)
	h.submitUsageRecordTask(func(ctx context.Context) {
		if err := h.gatewayService.RecordWebSearchUsage(ctx, &service.OpenAIRecordWebSearchUsageInput{
			Result:             result,
			APIKey:             currentAPIKey,
			User:               currentAPIKey.User,
			Account:            account,
			Subscription:       currentSubscription,
			InboundEndpoint:    GetInboundEndpoint(c),
			UpstreamEndpoint:   service.EndpointAlphaSearch,
			UserAgent:          userAgent,
			IPAddress:          clientIP,
			RequestPayloadHash: requestPayloadHash,
			APIKeyService:      h.apiKeyService,
		}); err != nil {
			logger.L().With(
				zap.String("component", "handler.openai_gateway.alpha_search"),
				zap.Int64("user_id", subject.UserID),
				zap.Int64("api_key_id", currentAPIKey.ID),
				zap.Any("group_id", currentAPIKey.GroupID),
				zap.Int64("account_id", account.ID),
			).Error("openai.alpha_search.record_usage_failed", zap.Error(err))
		}
	})
	reqLog.Debug("openai.alpha_search.completed", zap.Int64("account_id", account.ID), zap.Int("status_code", result.StatusCode))
}

func alphaSearchStatusCode(result *service.OpenAIAlphaSearchForwardResult) int {
	if result == nil {
		return 0
	}
	return result.StatusCode
}
