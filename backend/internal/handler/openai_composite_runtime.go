package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type openAICompositeRuntime struct {
	apiKey         *service.APIKey
	subscription   *service.UserSubscription
	displayModel   string
	runtimeModel   string
	selectionModel string
	body           []byte
	matched        bool
	targetGroup    *service.Group
	targetGroupID  int64
}

func resolveOpenAICompositeRuntime(
	c *gin.Context,
	gatewayService *service.OpenAIGatewayService,
	billingCacheService *service.BillingCacheService,
	reqLog *zap.Logger,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	body []byte,
	displayModel string,
	selectionModel string,
) (*openAICompositeRuntime, error) {
	result := &openAICompositeRuntime{
		apiKey:         apiKey,
		subscription:   subscription,
		displayModel:   strings.TrimSpace(displayModel),
		runtimeModel:   strings.TrimSpace(displayModel),
		selectionModel: strings.TrimSpace(selectionModel),
		body:           body,
	}
	if result.runtimeModel == "" {
		result.runtimeModel = result.selectionModel
	}
	if result.selectionModel == "" {
		result.selectionModel = result.runtimeModel
	}
	if apiKey == nil || apiKey.Group == nil || service.CanonicalizePlatformValue(apiKey.Group.Platform) != service.PlatformComposite {
		return result, nil
	}
	resolved, err := gatewayService.ResolveCompositeRouteRuntime(c.Request.Context(), apiKey, subscription, result.displayModel)
	if err != nil {
		if reqLog != nil {
			reqLog.Warn("openai.composite_route_resolve_failed", zap.Error(err), zap.String("model", result.displayModel))
		}
		return nil, err
	}
	if resolved == nil || !resolved.Matched {
		return result, nil
	}
	result.apiKey = resolved.APIKey
	result.subscription = resolved.Subscription
	if result.apiKey != nil && result.apiKey.Group != nil {
		selectedSubscription, subErr := loadSelectedSubscription(c.Request.Context(), result.apiKey, gatewayService.GetActiveSubscriptionForGroup)
		if subErr != nil {
			return nil, subErr
		}
		result.subscription = selectedSubscription
		if billingCacheService != nil {
			if billingErr := billingCacheService.CheckBillingEligibility(c.Request.Context(), result.apiKey.User, result.apiKey, result.apiKey.Group, selectedSubscription); billingErr != nil {
				return nil, billingErr
			}
		}
	}
	result.runtimeModel = firstNonEmptyHandlerString(resolved.RuntimeModelID, result.displayModel, result.selectionModel)
	result.selectionModel = result.runtimeModel
	result.matched = true
	result.targetGroup = resolved.TargetGroup
	result.targetGroupID = resolved.TargetGroupID
	if result.apiKey != nil && result.apiKey.Group != nil {
		applyOpenAIPlatformContext(c, result.apiKey.Group.Platform)
		applySelectedAPIKeyContext(c, result.apiKey, result.subscription)
		attachOpenAICompositeSelectionState(c, result.displayModel, result.runtimeModel, result.apiKey.Group)
	}
	if reqLog != nil {
		reqLog.Debug(
			"openai.composite_route_resolved",
			zap.Int64("parent_group_id", resolved.ParentGroupID),
			zap.Int64("target_group_id", result.targetGroupID),
			zap.String("display_model", result.displayModel),
			zap.String("runtime_model", result.runtimeModel),
		)
	}
	return result, nil
}

func attachOpenAICompositeSelectionState(c *gin.Context, displayModel string, runtimeModel string, group *service.Group) {
	if c == nil || c.Request == nil || group == nil || strings.TrimSpace(runtimeModel) == "" {
		return
	}
	ctx := c.Request.Context()
	state, _ := service.GatewayChannelStateFromContext(ctx)
	if state != nil {
		copyState := *state
		copyState.GroupID = group.ID
		copyState.Platform = group.Platform
		copyState.RequestedModel = strings.TrimSpace(displayModel)
		copyState.SelectionModel = strings.TrimSpace(runtimeModel)
		c.Request = c.Request.WithContext(service.WithGatewayChannelState(ctx, &copyState))
		return
	}
	c.Request = c.Request.WithContext(service.WithGatewayChannelState(ctx, &service.GatewayChannelState{
		GroupID:        group.ID,
		Platform:       group.Platform,
		RequestedModel: strings.TrimSpace(displayModel),
		SelectionModel: strings.TrimSpace(runtimeModel),
	}))
}

func compositeRouteErrorDetails(err error) (int, string, string) {
	switch {
	case errors.Is(err, service.ErrCompositeRouteNotFound):
		return http.StatusBadRequest, "invalid_request_error", "Requested model is not routed by the composite group"
	case errors.Is(err, service.ErrCompositeTargetGroupInvalid), errors.Is(err, service.ErrCompositeRouteInvalid), errors.Is(err, service.ErrCompositeRouteCycleForbidden), errors.Is(err, service.ErrCompositeGroupRequired):
		return http.StatusBadRequest, "invalid_request_error", "Composite route configuration is invalid"
	default:
		return http.StatusServiceUnavailable, "api_error", "Composite route target is temporarily unavailable"
	}
}
