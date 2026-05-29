package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

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
		Group:                   groupFromServiceBase(g),
		ModelRouting:            g.ModelRouting,
		ModelRoutingEnabled:     g.ModelRoutingEnabled,
		MCPXMLInject:            g.MCPXMLInject,
		DefaultMappedModel:      g.DefaultMappedModel,
		SupportedModelScopes:    g.SupportedModelScopes,
		AccountCount:            g.AccountCount,
		ActiveAccountCount:      g.ActiveAccountCount,
		RateLimitedAccountCount: g.RateLimitedAccountCount,
		AvailableAccountCount:   g.AvailableAccountCount,
		SortOrder:               g.SortOrder,
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
		IsExclusive:                     g.IsExclusive,
		Status:                          g.Status,
		SubscriptionType:                g.SubscriptionType,
		DailyLimitUSD:                   g.DailyLimitUSD,
		WeeklyLimitUSD:                  g.WeeklyLimitUSD,
		MonthlyLimitUSD:                 g.MonthlyLimitUSD,
		ImagePrice1K:                    g.ImagePrice1K,
		ImagePrice2K:                    g.ImagePrice2K,
		ImagePrice4K:                    g.ImagePrice4K,
		ImageProtocolMode:               g.ImageProtocolMode,
		ClaudeCodeOnly:                  g.ClaudeCodeOnly,
		FallbackGroupID:                 g.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest: g.FallbackGroupIDOnInvalidRequest,
		AllowMessagesDispatch:           g.AllowMessagesDispatch,
		GeminiMixedProtocolEnabled:      g.GeminiMixedProtocolEnabled,
		VisibleModelPatterns:            service.NormalizeGroupVisibleModelPatterns(g.VisibleModelPatterns),
		CreatedAt:                       g.CreatedAt,
		UpdatedAt:                       g.UpdatedAt,
	}
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
