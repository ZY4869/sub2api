package handler

import (
	"errors"
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

var errGatewayMessagesResponseHandled = errors.New("gateway messages response handled")

func (h *GatewayHandler) forwardGatewayMessagesClaude(c *gin.Context, req *gatewayMessagesRequest, route *gatewayMessagesRoute) bool {
	if !h.rebindGatewayMessagesRouteChannel(c, req, route) {
		return true
	}
	fallbackGroupID := gatewayMessagesFallbackGroupID(route.apiKey)
	fallbackUsed := false
	if h.gatewayService.IsSingleAntigravityAccountGroup(c.Request.Context(), route.apiKey.GroupID) {
		ctx := service.WithSingleAccountRetry(c.Request.Context(), true, h.metadataBridgeEnabled())
		c.Request = c.Request.WithContext(ctx)
	}

	for {
		fs := NewFailoverState(h.maxAccountSwitches, route.hasBoundSession)
		retryWithFallback := false
		for {
			selection, err := h.gatewayService.SelectAccountWithLoadAwareness(c.Request.Context(), route.apiKey.GroupID, route.sessionKey, route.runtimeSelectionModel, fs.FailedAccountIDs, req.parsedReq.MetadataUserID)
			if err != nil {
				retryGroup, done := h.handleGatewayMessagesSelectionError(c, req, route, fs, route.platform, err)
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

			queueRelease := h.acquireGatewayMessagesUserQueue(c, req, route, slot.account)
			req.parsedReq.OnUpstreamAccepted = queueRelease
			result, err := h.forwardGatewayMessagesClaudeAccount(c, req, route, fs, slot.account, queueRelease, slot.release)
			if errors.Is(err, errGatewayMessagesResponseHandled) {
				return true
			}
			if err != nil {
				retryFallback, retryGroup, done := h.handleGatewayMessagesClaudeForwardError(c, req, route, fs, slot.account, fallbackGroupID, fallbackUsed, err)
				if retryFallback {
					fallbackUsed = true
					retryWithFallback = true
					break
				}
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
		if !retryWithFallback {
			return false
		}
	}
}

func gatewayMessagesFallbackGroupID(apiKey *service.APIKey) *int64 {
	if apiKey == nil || apiKey.Group == nil {
		return nil
	}
	return apiKey.Group.FallbackGroupIDOnInvalidRequest
}

func (h *GatewayHandler) acquireGatewayMessagesUserQueue(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	account *service.Account,
) func() {
	var queueRelease func()
	umqMode := h.getUserMsgQueueMode(account, req.parsedReq)
	switch umqMode {
	case config.UMQModeSerialize:
		baseRPM := account.GetBaseRPM()
		release, qErr := h.userMsgQueueHelper.AcquireWithWait(c, account.ID, baseRPM, req.reqStream, &req.streamStarted, h.cfg.Gateway.UserMessageQueue.WaitTimeout(), req.reqLog)
		if qErr != nil {
			req.reqLog.Warn("gateway.umq_acquire_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID), zap.Error(qErr))
		} else {
			queueRelease = release
		}
	case config.UMQModeThrottle:
		baseRPM := account.GetBaseRPM()
		if tErr := h.userMsgQueueHelper.ThrottleWithPing(c, account.ID, baseRPM, req.reqStream, &req.streamStarted, h.cfg.Gateway.UserMessageQueue.WaitTimeout(), req.reqLog); tErr != nil {
			req.reqLog.Warn("gateway.umq_throttle_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID), zap.Error(tErr))
		}
	default:
		if umqMode != "" {
			req.reqLog.Warn("gateway.umq_unknown_mode", zap.String("mode", umqMode), zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID))
		}
	}
	return wrapReleaseOnDone(c.Request.Context(), queueRelease)
}

func (h *GatewayHandler) forwardGatewayMessagesClaudeAccount(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	fs *FailoverState,
	account *service.Account,
	queueRelease func(),
	accountRelease func(),
) (*service.ForwardResult, error) {
	requestCtx := c.Request.Context()
	if fs.SwitchCount > 0 {
		requestCtx = service.WithAccountSwitchCount(requestCtx, fs.SwitchCount, h.metadataBridgeEnabled())
	}
	if account.Platform == service.PlatformDeepSeek {
		if err := service.ValidateDeepSeekAnthropicMessagesBody(req.body); err != nil {
			if queueRelease != nil {
				queueRelease()
			}
			req.parsedReq.OnUpstreamAccepted = nil
			if accountRelease != nil {
				accountRelease()
			}
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
			h.handleStreamingAwareError(c, http.StatusBadRequest, "invalid_request_error", err.Error(), req.streamStarted)
			return nil, errGatewayMessagesResponseHandled
		}
	}

	var result *service.ForwardResult
	var err error
	if account.Platform == service.PlatformAntigravity && account.Type != service.AccountTypeAPIKey {
		result, err = h.antigravityGatewayService.Forward(requestCtx, c, account, req.body, route.hasBoundSession)
	} else {
		result, err = h.gatewayService.Forward(requestCtx, c, account, req.parsedReq)
	}
	if queueRelease != nil {
		queueRelease()
	}
	req.parsedReq.OnUpstreamAccepted = nil
	if accountRelease != nil {
		accountRelease()
	}
	return result, err
}

func (h *GatewayHandler) handleGatewayMessagesClaudeForwardError(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	fs *FailoverState,
	account *service.Account,
	fallbackGroupID *int64,
	fallbackUsed bool,
	err error,
) (bool, bool, bool) {
	if err == nil {
		return false, false, true
	}
	var betaBlockedErr *service.BetaBlockedError
	if errors.As(err, &betaBlockedErr) {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
		h.errorResponse(c, http.StatusBadRequest, "invalid_request_error", betaBlockedErr.Message)
		return false, false, true
	}
	var promptTooLongErr *service.PromptTooLongError
	if errors.As(err, &promptTooLongErr) {
		return h.handleGatewayMessagesPromptTooLong(c, req, route, account, fallbackGroupID, fallbackUsed, promptTooLongErr)
	}
	var failoverErr *service.UpstreamFailoverError
	if errors.As(err, &failoverErr) {
		action := fs.HandleFailoverError(c.Request.Context(), h.gatewayService, account.ID, account.Platform, failoverErr)
		switch action {
		case FailoverContinue:
			return false, false, false
		case FailoverExhausted:
			if excludeSelectedGroup(req.excludedGroupIDs, route.apiKey) {
				wroteFallback := h.ensureForwardErrorResponse(c, req.streamStarted)
				h.submitGatewayMessagesFailedUsage(c, req, route.apiKey, route.subscription, account, nil, err)
				req.reqLog.Error("gateway.forward_failed", append([]zap.Field{zap.Any("group_id", route.apiKey.GroupID)}, forwardFailedLogFields(account, wroteFallback, err)...)...)
				return false, false, true
			}
			h.submitGatewayMessagesFailedUsage(c, req, route.apiKey, route.subscription, account, fs.LastFailoverErr, err)
			h.handleFailoverExhausted(c, fs.LastFailoverErr, account.Platform, req.streamStarted)
			return false, false, true
		case FailoverCanceled:
			return false, false, true
		}
	}
	wroteFallback := h.ensureForwardErrorResponse(c, req.streamStarted)
	h.submitGatewayMessagesFailedUsage(c, req, route.apiKey, route.subscription, account, nil, err)
	req.reqLog.Error("gateway.forward_failed", append([]zap.Field{zap.Any("group_id", route.apiKey.GroupID)}, forwardFailedLogFields(account, wroteFallback, err)...)...)
	return false, false, true
}
