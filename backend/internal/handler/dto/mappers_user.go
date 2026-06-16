package dto

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

func UserFromServiceShallow(u *service.User) *User {
	if u == nil {
		return nil
	}
	return &User{
		ID:                                    u.ID,
		Email:                                 u.Email,
		Username:                              u.Username,
		Role:                                  u.Role,
		AdminFreeBilling:                      u.AdminFreeBilling,
		RequestDetailsReview:                  u.CanReviewRequestDetails(),
		UsageModelDisplayMode:                 u.EffectiveUsageModelDisplayMode(),
		VisualPresetPreference:                service.NormalizeVisualPresetPreference(u.VisualPresetPreference),
		AccountVisualPresetOverride:           service.NormalizeVisualPresetPreference(u.AccountVisualPresetOverride),
		AccountTodayStatsWindows:              service.NormalizeAccountTodayStatsWindows(u.AccountTodayStatsWindows),
		AccountTodayStatsCycleMode:            service.NormalizeAccountTodayStatsCycleMode(u.AccountTodayStatsCycleMode),
		AccountGroupDisplayMode:               service.NormalizeAccountGroupDisplayMode(u.AccountGroupDisplayMode),
		AccountStatusDisplayMode:              service.NormalizeAccountStatusDisplayMode(u.AccountStatusDisplayMode),
		APIKeyModelBindingMode:                u.EffectiveAPIKeyModelBindingMode(),
		ExternalModelCatalogViewMode:          service.NormalizeExternalModelCatalogViewMode(u.ExternalModelCatalogViewMode),
		EffectiveExternalModelCatalogViewMode: u.EffectiveExternalModelCatalogViewMode(),
		APIKeyAccessTimePolicy:                u.APIKeyAccessTimePolicy,
		GlobalRealtimeCountdownEnabled:        u.GlobalRealtimeCountdownEnabled,
		AccountRealtimeCountdownEnabled:       u.AccountRealtimeCountdownEnabled,
		Balance:                               u.Balance,
		Balances:                              cloneUsageCostByCurrency(u.Balances),
		Concurrency:                           u.Concurrency,
		Status:                                u.Status,
		AllowedGroups:                         u.AllowedGroups,
		CreatedAt:                             u.CreatedAt,
		UpdatedAt:                             u.UpdatedAt,
	}
}

func UserFromService(u *service.User) *User {
	if u == nil {
		return nil
	}
	out := UserFromServiceShallow(u)
	if len(u.APIKeys) > 0 {
		out.APIKeys = make([]APIKey, 0, len(u.APIKeys))
		for i := range u.APIKeys {
			k := u.APIKeys[i]
			out.APIKeys = append(out.APIKeys, *APIKeyFromService(&k))
		}
	}
	if len(u.Subscriptions) > 0 {
		out.Subscriptions = make([]UserSubscription, 0, len(u.Subscriptions))
		for i := range u.Subscriptions {
			s := u.Subscriptions[i]
			out.Subscriptions = append(out.Subscriptions, *UserSubscriptionFromService(&s))
		}
	}
	return out
}

// UserFromServiceAdmin converts a service User to DTO for admin users.
// It includes notes - user-facing endpoints must not use this.
func UserFromServiceAdmin(u *service.User) *AdminUser {
	if u == nil {
		return nil
	}
	base := UserFromService(u)
	if base == nil {
		return nil
	}
	return &AdminUser{
		User:             *base,
		Notes:            u.Notes,
		AdminFreeBilling: u.AdminFreeBilling,
		GroupRates:       u.GroupRates,
	}
}

func AuthIdentityFromService(identity *service.AuthIdentity) *AuthIdentity {
	if identity == nil {
		return nil
	}
	return &AuthIdentity{
		ID:             identity.ID,
		Provider:       identity.Provider,
		ProviderUserID: identity.ProviderUserID,
		Email:          identity.Email,
		EmailVerified:  identity.EmailVerified,
		DisplayName:    identity.DisplayName,
		AvatarURL:      identity.AvatarURL,
		CreatedAt:      identity.CreatedAt.Format(time.RFC3339),
		UpdatedAt:      identity.UpdatedAt.Format(time.RFC3339),
	}
}
