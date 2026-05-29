package handler

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) forwardGatewayMessagesGemini(c *gin.Context, req *gatewayMessagesRequest, route *gatewayMessagesRoute) bool {
	fs := NewFailoverState(h.maxAccountSwitchesGemini, route.hasBoundSession)
	if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), route.apiKey.GroupID) {
		ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
	}
	for {
		selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), route.apiKey.GroupID, route.sessionKey, route.runtimeSelectionModel, fs.FailedAccountIDs, "")
		if err != nil {
			retryGroup, done := h.handleGatewayMessagesSelectionError(c, req, route, fs, service.PlatformGemini, err)
			if retryGroup {
				return false
			}
			if done {
				return true
			}
			continue
		}

		slot := h.acquireGatewayMessagesAccountSlot(c, req, route, selection)
		switch slot.result {
		case gatewayMessagesAccountSlotRetryGroup:
			return false
		case gatewayMessagesAccountSlotStop:
			return true
		}

		result, err := h.forwardGatewayMessagesGeminiAccount(c, req, route, fs, slot.account)
		if slot.release != nil {
			slot.release()
		}
		if err != nil {
			retryGroup, done := h.handleGatewayMessagesGeminiForwardError(c, req, route, fs, slot.account, err)
			if retryGroup {
				return false
			}
			if done {
				return true
			}
			continue
		}
		h.finishGatewayMessagesSuccess(c, req, route, slot.account, result, fs.ForceCacheBilling)
		return true
	}
}

func (h *GatewayHandler) forwardGatewayMessagesGeminiAccount(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	fs *FailoverState,
	account *service.Account,
) (*service.ForwardResult, error) {
	requestCtx := c.Request.Context()
	if fs.SwitchCount > 0 {
		requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
	}
	if account.Platform == service.PlatformAntigravity {
		return h.antigravityGatewayService.ForwardGemini(requestCtx, c, account, req.reqModel, "generateContent", req.reqStream, req.body, route.hasBoundSession)
	}
	return h.geminiCompatService.Forward(requestCtx, c, account, req.body)
}

func (h *GatewayHandler) handleGatewayMessagesSelectionError(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	fs *FailoverState,
	exhaustedPlatform string,
	err error,
) (bool, bool) {
	if len(fs.FailedAccountIDs) == 0 {
		if excludeSelectedGroup(req.excludedGroupIDs, route.apiKey) {
			releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, route.apiKey)
			return true, false
		}
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
		h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts: "+err.Error(), req.streamStarted)
		return false, true
	}
	action := fs.HandleSelectionExhausted(c.Request.Context())
	switch action {
	case FailoverContinue:
		ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
		return false, false
	case FailoverCanceled:
		return false, true
	default:
		if excludeSelectedGroup(req.excludedGroupIDs, route.apiKey) {
			releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, route.apiKey)
			h.selectionAccountOrFail(c, nil, req.streamStarted)
			return false, true
		}
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
		if fs.LastFailoverErr != nil {
			h.handleFailoverExhausted(c, fs.LastFailoverErr, exhaustedPlatform, req.streamStarted)
		} else {
			h.handleFailoverExhaustedSimple(c, 502, req.streamStarted)
		}
		return false, true
	}
}

func (h *GatewayHandler) handleGatewayMessagesGeminiForwardError(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	fs *FailoverState,
	account *service.Account,
	err error,
) (bool, bool) {
	var failoverErr *service.UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, failoverErr)
		switch action {
		case FailoverContinue:
			return false, false
		case FailoverExhausted:
			if excludeSelectedGroup(req.excludedGroupIDs, route.apiKey) {
				wroteFallback := h.ensureForwardErrorResponse(c, req.streamStarted)
				h.submitGatewayMessagesFailedUsage(c, req, route.apiKey, route.subscription, account, nil, err)
				req.reqLog.Error("gateway.forward_failed", append([]zap.Field{zap.Any("group_id", route.apiKey.GroupID)}, forwardFailedLogFields(account, wroteFallback, err)...)...)
				return false, true
			}
			h.submitGatewayMessagesFailedUsage(c, req, route.apiKey, route.subscription, account, fs.LastFailoverErr, err)
			h.handleFailoverExhausted(c, fs.LastFailoverErr, service.PlatformGemini, req.streamStarted)
			return false, true
		case FailoverCanceled:
			return false, true
		}
	}
	wroteFallback := h.ensureForwardErrorResponse(c, req.streamStarted)
	h.submitGatewayMessagesFailedUsage(c, req, route.apiKey, route.subscription, account, nil, err)
	req.reqLog.Error("gateway.forward_failed", append([]zap.Field{zap.Any("group_id", route.apiKey.GroupID)}, forwardFailedLogFields(account, wroteFallback, err)...)...)
	return false, true
}

func (h *GatewayHandler) finishGatewayMessagesSuccess(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	account *service.Account,
	result *service.ForwardResult,
	forceCacheBilling bool,
) {
	if account.IsAnthropicOAuthOrSetupToken() && account.GetBaseRPM() > 0 {
		if err := h.gatewayService.IncrementAccountRPM(c.Request.Context(), account.ID); err != nil {
			req.reqLog.Warn("gateway.rpm_increment_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID), zap.Error(err))
		}
	}
	h.submitGatewayMessagesSuccessUsage(c, req, route, account, result, forceCacheBilling)
}
