package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func GroupFromServiceShallow(g *service.Group) *Group {
	if g == nil {
		return nil
	}
	out := groupFromServiceBase(g)
	return &out
}

func GroupFromService(g *service.Group) *Group {
	if g == nil {
		return nil
	}
	return GroupFromServiceShallow(g)
}

// GroupFromServiceAdmin converts a service Group to DTO for admin users.
// It includes internal fields like model_routing and account_count.
func GroupFromServiceAdmin(g *service.Group) *AdminGroup {
	if g == nil {
		return nil
	}
	out := &AdminGroup{
		Group:                       groupFromServiceBase(g),
		ModelRouting:                g.ModelRouting,
		ModelRoutingEnabled:         g.ModelRoutingEnabled,
		MCPXMLInject:                g.MCPXMLInject,
		DefaultMappedModel:          g.DefaultMappedModel,
		MaxReasoningEffort:          service.NormalizeOpenAIReasoningEffortSetting(g.MaxReasoningEffort),
		MaxReasoningEffortOverLimit: service.NormalizeReasoningEffortOverLimitAction(g.MaxReasoningEffortOverLimit),
		ForceOpenAIFast:             g.ForceOpenAIFast,
		FreeOpenAIFast:              g.FreeOpenAIFast,
		ReasoningEffortMappings:     service.NormalizeReasoningEffortMappings(g.ReasoningEffortMappings),
		SupportedModelScopes:        g.SupportedModelScopes,
		AccountCount:                g.AccountCount,
		ActiveAccountCount:          g.ActiveAccountCount,
		RateLimitedAccountCount:     g.RateLimitedAccountCount,
		AvailableAccountCount:       g.AvailableAccountCount,
		SortOrder:                   g.SortOrder,
		ImageBatchEnabled:           g.ImageBatchEnabled,
		ImageBatchAllowedProviders: service.NormalizeImageBatchAllowList(
			g.ImageBatchAllowedProviders,
		),
		ImageBatchAllowedModels:       service.NormalizeImageBatchAllowList(g.ImageBatchAllowedModels),
		ImageBatchMaxItems:            service.NormalizeImageBatchMaxItems(g.ImageBatchMaxItems),
		ImageBatchMaxDownloadBytes:    service.NormalizeImageBatchMaxDownloadBytes(g.ImageBatchMaxDownloadBytes),
		ImageBatchDownloadConcurrency: service.NormalizeImageBatchDownloadConcurrency(g.ImageBatchDownloadConcurrency),
		CompositeRoutes:               CompositeModelRoutesFromService(g.CompositeRoutes),
	}
	if len(g.AccountGroups) > 0 {
		out.AccountGroups = make([]AccountGroup, 0, len(g.AccountGroups))
		for i := range g.AccountGroups {
			ag := g.AccountGroups[i]
			out.AccountGroups = append(out.AccountGroups, *AccountGroupFromService(&ag))
		}
	}
	return out
}

func groupFromServiceBase(g *service.Group) Group {
	return Group{
		ID:                              g.ID,
		Name:                            g.Name,
		Description:                     g.Description,
		Platform:                        service.CanonicalizePlatformValue(g.Platform),
		Priority:                        g.Priority,
		RateMultiplier:                  g.RateMultiplier,
		ProfitControlEnabled:            g.ProfitControlEnabled,
		ProfitMinMargin:                 g.ProfitMinMargin,
		ProfitSafetyBuffer:              g.ProfitSafetyBuffer,
		PeakRateEnabled:                 g.PeakRateEnabled,
		PeakStart:                       g.PeakStart,
		PeakEnd:                         g.PeakEnd,
		PeakRateMultiplier:              g.PeakRateMultiplier,
		PeakTimezone:                    service.ServerPeakRateTimezoneName(),
		PeakUTCOffset:                   service.ServerPeakRateUTCOffset(time.Now()),
		IsExclusive:                     g.IsExclusive,
		Status:                          g.Status,
		SubscriptionType:                g.SubscriptionType,
		DailyLimitUSD:                   g.DailyLimitUSD,
		WeeklyLimitUSD:                  g.WeeklyLimitUSD,
		MonthlyLimitUSD:                 g.MonthlyLimitUSD,
		ImagePrice1K:                    g.ImagePrice1K,
		ImagePrice2K:                    g.ImagePrice2K,
		ImagePrice4K:                    g.ImagePrice4K,
		WebSearchPricePerCall:           g.WebSearchPricePerCall,
		ImageProtocolMode:               g.ImageProtocolMode,
		ClaudeCodeOnly:                  g.ClaudeCodeOnly,
		FallbackGroupID:                 g.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest: g.FallbackGroupIDOnInvalidRequest,
		AllowMessagesDispatch:           g.AllowMessagesDispatch,
		AllowLive:                       g.AllowLive,
		GeminiMixedProtocolEnabled:      g.GeminiMixedProtocolEnabled,
		VisibleModelPatterns:            service.NormalizeGroupVisibleModelPatterns(g.VisibleModelPatterns),
		CreatedAt:                       g.CreatedAt,
		UpdatedAt:                       g.UpdatedAt,
	}
}

func CompositeModelRouteFromService(route *service.CompositeModelRoute) *CompositeModelRoute {
	if route == nil {
		return nil
	}
	return &CompositeModelRoute{
		ID:             route.ID,
		ParentGroupID:  route.ParentGroupID,
		DisplayModelID: route.DisplayModelID,
		TargetGroupID:  route.TargetGroupID,
		TargetModelID:  route.TargetModelID,
		Priority:       route.Priority,
		Enabled:        route.Enabled,
		Notes:          route.Notes,
		CreatedAt:      route.CreatedAt,
		UpdatedAt:      route.UpdatedAt,
		TargetGroup:    GroupFromServiceShallow(route.TargetGroup),
	}
}

func CompositeModelRoutesFromService(routes []service.CompositeModelRoute) []CompositeModelRoute {
	out := make([]CompositeModelRoute, 0, len(routes))
	for i := range routes {
		out = append(out, *CompositeModelRouteFromService(&routes[i]))
	}
	return out
}

func AccountGroupFromService(ag *service.AccountGroup) *AccountGroup {
	if ag == nil {
		return nil
	}
	return &AccountGroup{
		AccountID: ag.AccountID,
		GroupID:   ag.GroupID,
		Priority:  ag.Priority,
		CreatedAt: ag.CreatedAt,
		Account:   AccountFromServiceShallow(ag.Account),
		Group:     GroupFromServiceShallow(ag.Group),
	}
}
