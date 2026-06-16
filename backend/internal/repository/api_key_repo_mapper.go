package repository

import (
	"encoding/json"
	dbent "github.com/Wei-Shaw/sub2api/ent"
	"github.com/Wei-Shaw/sub2api/internal/service"
)

func apiKeyEntityToService(m *dbent.APIKey) *service.APIKey {
	if m == nil {
		return nil
	}
	out := &service.APIKey{
		ID:                       m.ID,
		UserID:                   m.UserID,
		Key:                      m.Key,
		Name:                     m.Name,
		Deleted:                  m.DeletedAt != nil,
		ModelDisplayMode:         service.NormalizeAPIKeyModelDisplayMode(m.ModelDisplayMode),
		Status:                   m.Status,
		IPWhitelist:              m.IPWhitelist,
		IPBlacklist:              m.IPBlacklist,
		LastUsedAt:               m.LastUsedAt,
		CreatedAt:                m.CreatedAt,
		UpdatedAt:                m.UpdatedAt,
		GroupID:                  m.GroupID,
		ImageOnlyEnabled:         m.ImageOnlyEnabled,
		ImageCountBillingEnabled: m.ImageCountBillingEnabled,
		ImageMaxCount:            m.ImageMaxCount,
		ImageCountUsed:           m.ImageCountUsed,
		ImageCountWeights:        service.NormalizeAPIKeyImageCountWeights(m.ImageCountWeights),
		Quota:                    m.Quota,
		QuotaUsed:                m.QuotaUsed,
		QuotaUsedByCurrency:      service.CloneBillingCurrencyMap(m.QuotaUsedByCurrency),
		ExpiresAt:                m.ExpiresAt,
		StartsAt:                 m.StartsAt,
		AccessTimePolicy:         timeAccessPolicyFromMap(m.AccessTimePolicy),
		RateLimit5h:              m.RateLimit5h,
		RateLimit1d:              m.RateLimit1d,
		RateLimit7d:              m.RateLimit7d,
		Usage5h:                  m.Usage5h,
		Usage1d:                  m.Usage1d,
		Usage7d:                  m.Usage7d,
		Usage5hByCurrency:        service.CloneBillingCurrencyMap(m.Usage5hByCurrency),
		Usage1dByCurrency:        service.CloneBillingCurrencyMap(m.Usage1dByCurrency),
		Usage7dByCurrency:        service.CloneBillingCurrencyMap(m.Usage7dByCurrency),
		Window5hStart:            m.Window5hStart,
		Window1dStart:            m.Window1dStart,
		Window7dStart:            m.Window7dStart,
	}
	if m.Edges.User != nil {
		out.User = userEntityToService(m.Edges.User)
	}
	if m.Edges.Group != nil {
		out.Group = groupEntityToService(m.Edges.Group)
	}
	return out
}

func userEntityToService(u *dbent.User) *service.User {
	if u == nil {
		return nil
	}
	return &service.User{
		ID:                              u.ID,
		Email:                           u.Email,
		Username:                        u.Username,
		Notes:                           u.Notes,
		PasswordHash:                    u.PasswordHash,
		Role:                            u.Role,
		Balance:                         u.Balance,
		Balances:                        map[string]float64{service.ModelPricingCurrencyUSD: u.Balance},
		Concurrency:                     u.Concurrency,
		Status:                          u.Status,
		Deleted:                         u.DeletedAt != nil,
		AdminFreeBilling:                u.AdminFreeBilling,
		RequestDetailsReview:            u.RequestDetailsReview,
		GlobalRealtimeCountdownEnabled:  u.GlobalRealtimeCountdownEnabled,
		AccountRealtimeCountdownEnabled: u.AccountRealtimeCountdownEnabled,
		VisualPresetPreference: service.NormalizeVisualPresetPreference(
			u.VisualPresetPreference,
		),
		AccountVisualPresetOverride: service.NormalizeVisualPresetPreference(
			u.AccountVisualPresetOverride,
		),
		AccountTodayStatsWindows: service.NormalizeAccountTodayStatsWindows(
			u.AccountTodayStatsWindows,
		),
		AccountTodayStatsCycleMode: service.NormalizeAccountTodayStatsCycleMode(
			u.AccountTodayStatsCycleMode,
		),
		AccountGroupDisplayMode: service.NormalizeAccountGroupDisplayMode(
			u.AccountGroupDisplayMode,
		),
		AccountStatusDisplayMode: service.NormalizeAccountStatusDisplayMode(
			u.AccountStatusDisplayMode,
		),
		APIKeyModelBindingMode: service.NormalizeAPIKeyModelBindingMode(
			u.APIKeyModelBindingMode,
		),
		ExternalModelCatalogViewMode: service.NormalizeExternalModelCatalogViewMode(
			u.ExternalModelCatalogViewMode,
		),
		APIKeyAccessTimePolicy: timeAccessPolicyFromMap(u.APIKeyAccessTimePolicy),
		UsageModelDisplayMode: service.NormalizeUserUsageModelDisplayMode(
			u.UsageModelDisplayMode,
		),
		TotpSecretEncrypted: u.TotpSecretEncrypted,
		TotpEnabled:         u.TotpEnabled,
		TotpEnabledAt:       u.TotpEnabledAt,
		CreatedAt:           u.CreatedAt,
		UpdatedAt:           u.UpdatedAt,
	}
}

func parseBillingCurrencyJSONMap(raw []byte) map[string]float64 {
	if len(raw) == 0 {
		return nil
	}
	var values map[string]float64
	if err := json.Unmarshal(raw, &values); err != nil {
		return nil
	}
	return values
}

func groupEntityToService(g *dbent.Group) *service.Group {
	if g == nil {
		return nil
	}
	return &service.Group{
		ID:                              g.ID,
		Name:                            g.Name,
		Description:                     derefString(g.Description),
		Platform:                        service.CanonicalizePlatformValue(g.Platform),
		RateMultiplier:                  g.RateMultiplier,
		IsExclusive:                     g.IsExclusive,
		Status:                          g.Status,
		Hydrated:                        true,
		SubscriptionType:                g.SubscriptionType,
		DailyLimitUSD:                   g.DailyLimitUsd,
		WeeklyLimitUSD:                  g.WeeklyLimitUsd,
		MonthlyLimitUSD:                 g.MonthlyLimitUsd,
		ImagePrice1K:                    g.ImagePrice1k,
		ImagePrice2K:                    g.ImagePrice2k,
		ImagePrice4K:                    g.ImagePrice4k,
		ImageProtocolMode:               g.ImageProtocolMode,
		DefaultValidityDays:             g.DefaultValidityDays,
		ClaudeCodeOnly:                  g.ClaudeCodeOnly,
		FallbackGroupID:                 g.FallbackGroupID,
		FallbackGroupIDOnInvalidRequest: g.FallbackGroupIDOnInvalidRequest,
		ModelRouting:                    g.ModelRouting,
		ModelRoutingEnabled:             g.ModelRoutingEnabled,
		GeminiMixedProtocolEnabled:      g.GeminiMixedProtocolEnabled,
		MCPXMLInject:                    g.McpXMLInject,
		SupportedModelScopes:            g.SupportedModelScopes,
		SortOrder:                       g.SortOrder,
		Priority:                        g.Priority,
		AllowMessagesDispatch:           g.AllowMessagesDispatch,
		DefaultMappedModel:              g.DefaultMappedModel,
		CreatedAt:                       g.CreatedAt,
		UpdatedAt:                       g.UpdatedAt,
	}
}

func derefString(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
