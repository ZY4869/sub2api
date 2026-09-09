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
	openAICompatiblePlatforms     = []string{service.PlatformOpenAI, service.PlatformKimi, service.PlatformComposite}
	openAITextCompatiblePlatforms = []string{service.PlatformOpenAI, service.PlatformDeepSeek, service.PlatformKimi, service.PlatformOpenRouter, service.PlatformComposite}
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

func propagateSelectedBillingHold(source *service.APIKey, selected *service.APIKey) {
	if source == nil || selected == nil || source == selected || selected.BillingHold == nil {
		return
	}
	source.BillingHold = selected.BillingHold
}

func isGroupExcluded(apiKey *service.APIKey, excludedGroupIDs map[int64]struct{}) bool {
	if apiKey == nil || apiKey.GroupID == nil || len(excludedGroupIDs) == 0 {
		return false
	}
	_, excluded := excludedGroupIDs[*apiKey.GroupID]
	return excluded
}

func enforcePublicCatalogBindingGroup(ctx context.Context, apiKey *service.APIKey, excludedGroupIDs map[int64]struct{}) (*service.APIKey, bool, error) {
	entry, ok := service.PublishedPublicCatalogEntryFromContext(ctx)
	if !ok || entry == nil || entry.BindingGroupID <= 0 {
		return nil, false, nil
	}
	if excludedGroupIDs != nil {
		if _, excluded := excludedGroupIDs[entry.BindingGroupID]; excluded {
			return nil, true, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")
		}
	}
	for _, binding := range service.APIKeyBindingsForSelection(apiKey) {
		if binding.GroupID != entry.BindingGroupID {
			continue
		}
		if !apiKey.UserCanAccessGroup(binding.Group) {
			return nil, true, infraerrors.Forbidden("GROUP_ACCESS_DENIED", "api key is not allowed to access this group")
		}
		selectedAPIKey := service.CloneAPIKeyWithSelectedGroup(apiKey, &binding)
		return selectedAPIKey, true, nil
	}
	return nil, true, infraerrors.BadRequest("PUBLIC_MODEL_NOT_AVAILABLE", service.PublicCatalogModelUnavailableMessage)
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
	if selectedAPIKey, handled, err := enforcePublicCatalogBindingGroup(c.Request.Context(), apiKey, excludedGroupIDs); handled {
		if err != nil {
			return nil, nil, err
		}
		selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
		if err != nil {
			return nil, nil, err
		}
		if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
			return nil, nil, err
		}
		propagateSelectedBillingHold(apiKey, selectedAPIKey)
		applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
		return selectedAPIKey, selectedSubscription, nil
	}
	if !multiGroupRoutingEnabled(c.Request.Context(), apiKey, settingService) {
		if isGroupExcluded(apiKey, excludedGroupIDs) {
			return nil, nil, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")
		}
		if billingCacheService != nil {
			if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
				return nil, nil, err
			}
		}
		return apiKey, subscription, nil
	}
	if gatewayService != nil {
		candidates := gatewayService.ResolveAPIKeyVisibleModelCandidates(c.Request.Context(), apiKey, "", model)
		if len(candidates) > 0 {
			c.Request = c.Request.WithContext(service.WithVisibleModelCandidates(c.Request.Context(), candidates...))
		}
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
	propagateSelectedBillingHold(apiKey, selectedAPIKey)
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
	if selectedAPIKey, handled, err := enforcePublicCatalogBindingGroup(c.Request.Context(), apiKey, excludedGroupIDs); handled {
		if err != nil {
			return nil, nil, err
		}
		selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
		if err != nil {
			return nil, nil, err
		}
		if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
			return nil, nil, err
		}
		propagateSelectedBillingHold(apiKey, selectedAPIKey)
		applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
		return selectedAPIKey, selectedSubscription, nil
	}
	if !multiGroupRoutingEnabled(c.Request.Context(), apiKey, settingService) {
		if isGroupExcluded(apiKey, excludedGroupIDs) {
			return nil, nil, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")
		}
		if billingCacheService != nil {
			if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
				return nil, nil, err
			}
		}
		return apiKey, subscription, nil
	}
	if gatewayService != nil {
		candidates := gatewayService.ResolveAPIKeyVisibleModelCandidates(c.Request.Context(), apiKey, service.OpenAIPlatformFromContext(c.Request.Context()), model)
		if len(candidates) > 0 {
			c.Request = c.Request.WithContext(service.WithVisibleModelCandidates(c.Request.Context(), candidates...))
		}
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
	propagateSelectedBillingHold(apiKey, selectedAPIKey)
	applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
	return selectedAPIKey, selectedSubscription, nil
}

// resolveSelectedOpenAIEndpointCapability selects an OpenAI binding by
// protocol capability rather than treating the capability name as a model.
// Composite bindings remain model-routed and are resolved by the caller after
// this function returns.
func resolveSelectedOpenAIEndpointCapability(
	c *gin.Context,
	settingService *service.SettingService,
	gatewayService *service.OpenAIGatewayService,
	billingCacheService *service.BillingCacheService,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	requestedModel string,
	capability service.OpenAIEndpointCapability,
	excludedGroupIDs map[int64]struct{},
) (*service.APIKey, *service.UserSubscription, error) {
	if apiKey == nil {
		return nil, nil, service.ErrNoAvailableGroup
	}
	if selectedAPIKey, handled, err := enforcePublicCatalogBindingGroup(c.Request.Context(), apiKey, excludedGroupIDs); handled {
		if err != nil {
			return nil, nil, err
		}
		selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
		if err != nil {
			return nil, nil, err
		}
		if billingCacheService != nil {
			if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
				return nil, nil, err
			}
		}
		propagateSelectedBillingHold(apiKey, selectedAPIKey)
		applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
		return selectedAPIKey, selectedSubscription, nil
	}
	if !multiGroupRoutingEnabled(c.Request.Context(), apiKey, settingService) {
		if isGroupExcluded(apiKey, excludedGroupIDs) {
			return nil, nil, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")
		}
		if billingCacheService != nil {
			if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
				return nil, nil, err
			}
		}
		return apiKey, subscription, nil
	}
	if gatewayService == nil {
		return nil, nil, service.ErrNoAvailableGroup
	}
	binding, err := gatewayService.SelectGroupForOpenAIEndpointCapability(
		c.Request.Context(), apiKey, openAICompatiblePlatforms, requestedModel, capability, excludedGroupIDs,
	)
	if err != nil {
		return nil, nil, err
	}
	selectedAPIKey := service.CloneAPIKeyWithSelectedGroup(apiKey, binding)
	selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
	if err != nil {
		return nil, nil, err
	}
	if billingCacheService != nil {
		if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
			return nil, nil, err
		}
	}
	propagateSelectedBillingHold(apiKey, selectedAPIKey)
	applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
	return selectedAPIKey, selectedSubscription, nil
}

// resolveSelectedOpenAITransportCapability selects a binding that has a
// schedulable account supporting both the requested model and transport. It
// is used by Responses WebSocket ingress so group selection cannot stop at a
// group whose accounts are HTTP-only or otherwise incompatible.
func resolveSelectedOpenAITransportCapability(
	c *gin.Context,
	settingService *service.SettingService,
	gatewayService *service.OpenAIGatewayService,
	billingCacheService *service.BillingCacheService,
	apiKey *service.APIKey,
	subscription *service.UserSubscription,
	requestedModel string,
	requiredTransport service.OpenAIUpstreamTransport,
	requiredCapability service.OpenAIEndpointCapability,
	excludedGroupIDs map[int64]struct{},
) (*service.APIKey, *service.UserSubscription, error) {
	if apiKey == nil {
		return nil, nil, service.ErrNoAvailableGroup
	}
	if selectedAPIKey, handled, err := enforcePublicCatalogBindingGroup(c.Request.Context(), apiKey, excludedGroupIDs); handled {
		if err != nil {
			return nil, nil, err
		}
		if gatewayService == nil || selectedAPIKey.SelectedGroupBinding == nil {
			return nil, nil, service.ErrNoAvailableGroup
		}
		available, checkErr := gatewayService.GroupSupportsOpenAITransportCapability(
			c.Request.Context(), selectedAPIKey.SelectedGroupBinding, requestedModel, requiredTransport, requiredCapability,
		)
		if checkErr != nil {
			return nil, nil, checkErr
		}
		if available {
			selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
			if err != nil {
				return nil, nil, err
			}
			if billingCacheService != nil {
				if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
					return nil, nil, err
				}
			}
			propagateSelectedBillingHold(apiKey, selectedAPIKey)
			applySelectedAPIKeyContext(c, selectedAPIKey, selectedSubscription)
			return selectedAPIKey, selectedSubscription, nil
		}
		if excludedGroupIDs != nil {
			excludedGroupIDs[selectedAPIKey.SelectedGroupBinding.GroupID] = struct{}{}
		}
	}
	if gatewayService == nil {
		return nil, nil, service.ErrNoAvailableGroup
	}
	if !multiGroupRoutingEnabled(c.Request.Context(), apiKey, settingService) {
		if isGroupExcluded(apiKey, excludedGroupIDs) {
			return nil, nil, infraerrors.ServiceUnavailable("GROUP_EXHAUSTED", "all accounts in the group have been exhausted")
		}
		bindings := service.APIKeyBindingsForSelection(apiKey)
		if len(bindings) == 0 {
			return nil, nil, service.ErrNoAvailableGroup
		}
		available, err := gatewayService.GroupSupportsOpenAITransportCapability(
			c.Request.Context(), &bindings[0], requestedModel, requiredTransport, requiredCapability,
		)
		if err != nil {
			return nil, nil, err
		}
		if !available {
			return nil, nil, service.ErrNoAvailableGroup
		}
		if billingCacheService != nil {
			if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), apiKey.User, apiKey, apiKey.Group, subscription); err != nil {
				return nil, nil, err
			}
		}
		return apiKey, subscription, nil
	}
	binding, err := gatewayService.SelectGroupForOpenAITransportCapability(
		c.Request.Context(), apiKey, openAICompatiblePlatforms, requestedModel,
		requiredTransport, requiredCapability, excludedGroupIDs,
	)
	if err != nil {
		return nil, nil, err
	}
	selectedAPIKey := service.CloneAPIKeyWithSelectedGroup(apiKey, binding)
	selectedSubscription, err := loadSelectedSubscription(c.Request.Context(), selectedAPIKey, gatewayService.GetActiveSubscriptionForGroup)
	if err != nil {
		return nil, nil, err
	}
	if billingCacheService != nil {
		if err := billingCacheService.CheckBillingEligibility(c.Request.Context(), selectedAPIKey.User, selectedAPIKey, selectedAPIKey.Group, selectedSubscription); err != nil {
			return nil, nil, err
		}
	}
	propagateSelectedBillingHold(apiKey, selectedAPIKey)
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

// isGroupSelectionExhaustedError 判断选组失败是否属于"分组已被排除/耗尽"类错误。
// 只有这类错误才应该用最后一次上游 failover 错误向客户端映射真实状态码；
// 余额、订阅、权限等业务错误必须保留原始错误码与文案，不得被上游错误覆盖。
func isGroupSelectionExhaustedError(err error) bool {
	if err == nil {
		return false
	}
	switch infraerrors.FromError(err).Reason {
	case "GROUP_EXHAUSTED", "NO_AVAILABLE_GROUP":
		return true
	default:
		return false
	}
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
