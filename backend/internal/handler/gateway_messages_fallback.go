package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func (h *GatewayHandler) handleGatewayMessagesPromptTooLong(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	account *service.Account,
	fallbackGroupID *int64,
	fallbackUsed bool,
	promptTooLongErr *service.PromptTooLongError,
) (bool, bool, bool) {
	req.reqLog.Warn("gateway.prompt_too_long_from_antigravity", zap.Any("current_group_id", route.apiKey.GroupID), zap.Any("fallback_group_id", fallbackGroupID), zap.Bool("fallback_used", fallbackUsed))
	if !fallbackUsed && fallbackGroupID != nil && *fallbackGroupID > 0 {
		return h.tryGatewayMessagesPromptTooLongFallback(c, req, route, account, *fallbackGroupID, promptTooLongErr)
	}
	_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
	releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
	return false, false, true
}

func (h *GatewayHandler) tryGatewayMessagesPromptTooLongFallback(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	account *service.Account,
	fallbackGroupID int64,
	promptTooLongErr *service.PromptTooLongError,
) (bool, bool, bool) {
	fallbackGroup, err := h.gatewayService.ResolveGroupByID(c.Request.Context(), fallbackGroupID)
	if err != nil {
		req.reqLog.Warn("gateway.resolve_fallback_group_failed", zap.Int64("fallback_group_id", fallbackGroupID), zap.Error(err))
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
		_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
		return false, false, true
	}
	if fallbackGroup.Platform != service.PlatformAnthropic || fallbackGroup.SubscriptionType == service.SubscriptionTypeSubscription || fallbackGroup.FallbackGroupIDOnInvalidRequest != nil {
		req.reqLog.Warn("gateway.fallback_group_invalid", zap.Int64("fallback_group_id", fallbackGroup.ID), zap.String("fallback_platform", fallbackGroup.Platform), zap.String("fallback_subscription_type", fallbackGroup.SubscriptionType))
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
		_ = h.antigravityGatewayService.WriteMappedClaudeError(c, account, promptTooLongErr.StatusCode, promptTooLongErr.RequestID, promptTooLongErr.Body)
		return false, false, true
	}
	fallbackAPIKey := cloneAPIKeyWithGroup(route.apiKey, fallbackGroup)
	if err := h.billingCacheService.CheckBillingEligibility(c.Request.Context(), fallbackAPIKey.User, fallbackAPIKey, fallbackGroup, nil); err != nil {
		releaseHeldBillingHold(c.Request.Context(), h.apiKeyService, route.apiKey)
		status, code, message := billingErrorDetails(err)
		h.handleStreamingAwareError(c, status, code, message, req.streamStarted)
		return false, false, true
	}
	ctx := context.WithValue(c.Request.Context(), ctxkey.ForcePlatform, "")
	c.Request = c.Request.WithContext(ctx)
	route.apiKey = fallbackAPIKey
	route.subscription = nil
	return true, false, false
}
