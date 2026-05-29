package handler

import (
	"net/http"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) acquireGatewayMessagesAccountSlot(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	selection *service.AccountSelectionResult,
) gatewayMessagesAccountSlot {
	account, ok := h.selectionAccountOrFail(c, selection, req.streamStarted)
	if !ok {
		return gatewayMessagesAccountSlot{result: gatewayMessagesAccountSlotStop}
	}
	setOpsSelectedAccountDetails(c, account)
	setOpsEndpointContext(c, account.GetMappedModel(route.runtimeSelectionModel), service.RequestTypeFromLegacy(req.reqStream, false))
	if account.IsInterceptWarmupEnabled() {
		interceptType := detectInterceptType(req.body, req.reqModel, req.parsedReq.MaxTokens, req.reqStream, req.isClaudeCodeClient)
		if interceptType != InterceptTypeNone {
			if selection.Acquired && selection.ReleaseFunc != nil {
				selection.ReleaseFunc()
			}
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
			if req.reqStream {
				sendMockInterceptStream(c, req.reqModel, interceptType)
			} else {
				sendMockInterceptResponse(c, req.reqModel, interceptType)
			}
			return gatewayMessagesAccountSlot{result: gatewayMessagesAccountSlotStop}
		}
	}

	accountReleaseFunc := selection.ReleaseFunc
	if !selection.Acquired {
		if selection.WaitPlan == nil {
			if excludeSelectedGroup(req.excludedGroupIDs, route.apiKey) {
				releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, route.apiKey)
				return gatewayMessagesAccountSlot{result: gatewayMessagesAccountSlotRetryGroup}
			}
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
			h.handleStreamingAwareError(c, http.StatusServiceUnavailable, "api_error", "No available accounts", req.streamStarted)
			return gatewayMessagesAccountSlot{result: gatewayMessagesAccountSlotStop}
		}

		accountWaitCounted := false
		canWait, err := h.concurrencyHelper.IncrementAccountWaitCount(c.Request.Context(), account.ID, selection.WaitPlan.MaxWaiting)
		if err != nil {
			req.reqLog.Warn("gateway.account_wait_counter_increment_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID), zap.Error(err))
		} else if !canWait {
			req.reqLog.Info("gateway.account_wait_queue_full", zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID), zap.Int("max_waiting", selection.WaitPlan.MaxWaiting))
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
			h.handleStreamingAwareError(c, http.StatusTooManyRequests, "rate_limit_error", "Too many pending requests, please retry later", req.streamStarted)
			return gatewayMessagesAccountSlot{result: gatewayMessagesAccountSlotStop}
		}
		if err == nil && canWait {
			accountWaitCounted = true
		}
		releaseWait := func() {
			if accountWaitCounted {
				h.concurrencyHelper.DecrementAccountWaitCount(c.Request.Context(), account.ID)
				accountWaitCounted = false
			}
		}
		accountReleaseFunc, err = h.concurrencyHelper.AcquireAccountSlotWithWaitTimeout(c, account.ID, selection.WaitPlan.MaxConcurrency, selection.WaitPlan.Timeout, req.reqStream, &req.streamStarted)
		if err != nil {
			req.reqLog.Warn("gateway.account_slot_acquire_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID), zap.Error(err))
			releaseWait()
			releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
			h.handleConcurrencyError(c, err, "account", req.streamStarted)
			return gatewayMessagesAccountSlot{result: gatewayMessagesAccountSlotStop}
		}
		releaseWait()
		if err := h.gatewayService.BindStickySession(c.Request.Context(), route.apiKey.GroupID, route.sessionKey, account.ID); err != nil {
			req.reqLog.Warn("gateway.bind_sticky_session_failed", zap.Int64("account_id", account.ID), zap.Any("group_id", route.apiKey.GroupID), zap.Error(err))
		}
	}
	return gatewayMessagesAccountSlot{
		account: account,
		release: wrapReleaseOnDone(c.Request.Context(), accountReleaseFunc),
		result:  gatewayMessagesAccountSlotReady,
	}
}
