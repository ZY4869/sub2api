package handler

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// maxGatewayMessagesGroupSwitches 单次请求内跨分组 failover 的次数上限，
// 防止分组解析异常时外层循环无法收敛。
const maxGatewayMessagesGroupSwitches = 8

// hasAlternateGatewayMessagesGroup 判断排除掉当前分组后，该 API Key 是否还有别的分组可以尝试。
// 公开模型目录会把请求钉死在绑定分组上，不参与跨分组 failover。
func (h *GatewayHandler) hasAlternateGatewayMessagesGroup(c *gin.Context, req *gatewayMessagesRequest, apiKey *service.APIKey) bool {
	if req.publicCatalogEntry != nil || apiKey == nil || apiKey.GroupID == nil {
		return false
	}
	if !multiGroupRoutingEnabled(c.Request.Context(), req.apiKey, h.settingService) {
		return false
	}
	for _, binding := range service.APIKeyBindingsForSelection(req.apiKey) {
		if binding.GroupID == *apiKey.GroupID {
			continue
		}
		if _, excluded := req.excludedGroupIDs[binding.GroupID]; excluded {
			continue
		}
		return true
	}
	return false
}

// retryNextGatewayMessagesGroup 记录跨分组 failover 状态，排除当前分组并判断是否还值得重试下一个分组。
// 返回 true 时调用方应交还 runGatewayMessages 重新选组；
// 返回 false 表示没有分组可换，调用方应直接用 handleFailoverExhausted 映射真实上游错误。
func (h *GatewayHandler) retryNextGatewayMessagesGroup(
	c *gin.Context,
	req *gatewayMessagesRequest,
	route *gatewayMessagesRoute,
	failoverErr *service.UpstreamFailoverError,
	platform string,
) bool {
	if failoverErr != nil {
		req.lastFailoverErr = failoverErr
		req.lastFailoverPlatform = platform
	}
	if req.groupSwitchCount >= maxGatewayMessagesGroupSwitches {
		return false
	}
	if !h.hasAlternateGatewayMessagesGroup(c, req, route.apiKey) {
		return false
	}
	if !excludeSelectedGroup(req.excludedGroupIDs, route.apiKey) {
		return false
	}
	req.groupSwitchCount++
	releaseHeldBillingHoldBeforeRetry(c.Request.Context(), h.apiKeyService, route.apiKey)
	fields := []zap.Field{
		zap.Any("group_id", route.apiKey.GroupID),
		zap.Int("group_switch_count", req.groupSwitchCount),
		zap.Int("group_switch_max", maxGatewayMessagesGroupSwitches),
	}
	if failoverErr != nil {
		fields = append(fields, zap.Int("upstream_status", failoverErr.StatusCode))
	}
	req.reqLog.Warn("gateway.failover_switch_group", fields...)
	return true
}

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
