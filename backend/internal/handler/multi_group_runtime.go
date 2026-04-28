package handler

import (
	"context"
	"errors"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

type gatewayChannelStateResolver interface {
	ResolveChannelState(ctx context.Context, group *service.Group, requestedModel string) (*service.GatewayChannelState, error)
}

var (
	openAICompatiblePlatforms     = []string{service.PlatformOpenAI, service.PlatformCopilot}
	openAITextCompatiblePlatforms = []string{service.PlatformOpenAI, service.PlatformCopilot, service.PlatformDeepSeek}
	gatewayCompatiblePlatforms    = []string{
		service.PlatformAnthropic,
		service.PlatformDeepSeek,
		service.PlatformGemini,
		service.PlatformAntigravity,
		service.PlatformKiro,
	}
	geminiCompatiblePlatforms = []string{service.PlatformGemini, service.PlatformAntigravity}
	grokCompatiblePlatforms   = []string{service.PlatformGrok}
)

func multiGroupRoutingEnabled(ctx context.Context, apiKey *service.APIKey, settingService *service.SettingService) bool {
	if apiKey == nil || len(apiKey.GroupBindings) <= 1 {
		return false
	}
	if settingService == nil {
		return true
	}
	return settingService.IsMultiGroupRoutingEnabled(ctx)
}

func applySelectedAPIKeyContext(c *gin.Context, apiKey *service.APIKey, subscription *service.UserSubscription) {
	if c == nil || c.Request == nil || apiKey == nil {
		return
	}
	c.Set(string(middleware2.ContextKeyAPIKey), apiKey)
	c.Set(string(middleware2.ContextKeySubscription), subscription)

	ctx := c.Request.Context()
	if len(apiKey.GroupBindings) > 0 {
		groups := make([]*service.Group, 0, len(apiKey.GroupBindings))
		for _, binding := range apiKey.GroupBindings {
			if service.IsGroupContextValid(binding.Group) {
				groups = append(groups, binding.Group)
			}
		}
		if len(groups) > 0 {
			ctx = context.WithValue(ctx, ctxkey.Groups, groups)
		}
	}
	if service.IsGroupContextValid(apiKey.Group) {
		ctx = context.WithValue(ctx, ctxkey.Group, apiKey.Group)
	}
	c.Request = c.Request.WithContext(ctx)
}

func isGroupExcluded(apiKey *service.APIKey, excludedGroupIDs map[int64]struct{}) bool {
	if apiKey == nil || apiKey.GroupID == nil || len(excludedGroupIDs) == 0 {
		return false
	}
	_, excluded := excludedGroupIDs[*apiKey.GroupID]
	return excluded
}

func resolveSelectedGatewayAPIKey(
	c *gin.Context,
	settingService *service.SettingService,
	gatewayService *service.GatewayService,
	billingCacheService *service.BillingCacheService,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	model string,
	allowedPlatforms []string,
	excludedGroupIDs map[int64]struct{},
) (*service.APIKey, *service.UserSubscription, error) {
	if !multiGroupRoutingEnabled(c.Request.Context(), apiKey, settingService) {
		if isGroupExcluded(apiKey, excludedGroupIDs) {
			return nil, nil, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")
		}
		return apiKey, subscription, nil
	}
	binding, err := gatewayService.SelectGroupForAllowedPlatforms(c.Request.Context(), apiKey, allowedPlatforms, model, excludedGroupIDs)
	if err != nil {
		return nil, nil, err
	}
	selectedAPIKey := service.CloneAPIKeyWithSelectedGroup(apiKey, binding)
	selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
	if err != nil {
		return nil, nil, err
	}
	if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
		return nil, nil, err
	}
	applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
	return selectedAPIKey, selectedSubscription, nil
}

func resolveSelectedOpenAIAPIKey(
	c *gin.Context,
	settingService *service.SettingService,
	gatewayService *service.OpenAIGatewayService,
	billingCacheService *service.BillingCacheService,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	model string,
	allowedPlatforms []string,
	excludedGroupIDs map[int64]struct{},
) (*service.APIKey, *service.UserSubscription, error) {
	if !multiGroupRoutingEnabled(c.Request.Context(), apiKey, settingService) {
		if isGroupExcluded(apiKey, excludedGroupIDs) {
			return nil, nil, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")
		}
		return apiKey, subscription, nil
	}
	binding, err := gatewayService.SelectGroupForAllowedPlatforms(c.Request.Context(), apiKey, allowedPlatforms, model, excludedGroupIDs)
	if err != nil {
		return nil, nil, err
	}
	selectedAPIKey := service.CloneAPIKeyWithSelectedGroup(apiKey, binding)
	selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
	if err != nil {
		return nil, nil, err
	}
	if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
		return nil, nil, err
	}
	applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
	return selectedAPIKey, selectedSubscription, nil
}

func loadSelectedSubscription(
	ctx context.Context,
	apiKey *service.APIKey,
	loader func(context.Context, int64, int64) (*service.UserSubscription, error),
) (*service.UserSubscription, error) {
	if apiKey == nil || apiKey.Group == nil || !apiKey.Group.IsSubscriptionType() || loader == nil {
		return nil, nil
	}
	userID := apiKey.UserID
	if apiKey.User != nil && apiKey.User.ID > 0 {
		userID = apiKey.User.ID
	}
	if apiKey.GroupID == nil {
		return nil, infraerrors.BadRequest("INVALID_GROUP_BINDING", "selected api key group is missing group id")
	}
	subscription, err := loader(ctx, userID, *apiKey.GroupID)
	if err != nil {
		if errors.Is(err, service.ErrSubscriptionNotFound) {
			return nil, infraerrors.BadRequest("SUBSCRIPTION_REQUIRED", "user does not have an active subscription for this group")
		}
		return nil, err
	}
	return subscription, nil
}

func groupSelectionErrorDetails(err error) (int, string, string) {
	if err == nil {
		return 500, "api_error", "internal error"
	}
	appErr := infraerrors.FromError(err)
	code := appErr.Reason
	if code == "" {
		code = "api_error"
	}
	return int(appErr.Code), code, appErr.Message
}

func excludeSelectedGroup(excludedGroupIDs map[int64]struct{}, apiKey *service.APIKey) bool {
	if excludedGroupIDs == nil || apiKey == nil || apiKey.GroupID == nil {
		return false
	}
	excludedGroupIDs[*apiKey.GroupID] = struct{}{}
	return true
}

func bindGatewayChannelState(
	c *gin.Context,
	resolver gatewayChannelStateResolver,
	group *service.Group,
	requestedModel string,
) (string, *service.GatewayChannelState, error) {
	if c == nil || c.Request == nil || resolver == nil || group == nil {
		return requestedModel, nil, nil
	}

	state, err := resolver.ResolveChannelState(c.Request.Context(), group, requestedModel)
	if err != nil {
		return "", nil, err
	}

	ctx := c.Request.Context()
	if state != nil {
		ctx = service.WithGatewayChannelState(ctx, state)
	}
	c.Request = c.Request.WithContext(ctx)
	if state != nil && strings.TrimSpace(state.SelectionModel) != "" {
		return state.SelectionModel, state, nil
	}
	return requestedModel, state, nil
}

func reattachGatewayChannelState(ctx context.Context, state *service.GatewayChannelState) context.Context {
	if state == nil {
		return ctx
	}
	return service.WithGatewayChannelState(ctx, state)
}
