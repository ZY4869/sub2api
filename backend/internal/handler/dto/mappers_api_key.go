package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

func APIKeyFromService(k *service.APIKey) *APIKey {
	if k == nil {
		return nil
	}
	out := &APIKey{
		ID:                       k.ID,
		UserID:                   k.UserID,
		Key:                      k.Key,
		Name:                     k.Name,
		Deleted:                  k.Deleted,
		ModelDisplayMode:         k.EffectiveModelDisplayMode(),
		GroupID:                  k.GroupID,
		Status:                   k.Status,
		IPWhitelist:              k.IPWhitelist,
		IPBlacklist:              k.IPBlacklist,
		LastUsedAt:               k.LastUsedAt,
		ImageOnlyEnabled:         k.ImageOnlyEnabled,
		ImageCountBillingEnabled: k.ImageCountBillingEnabled,
		ImageMaxCount:            k.ImageMaxCount,
		ImageCountUsed:           k.ImageCountUsed,
		ImageCountWeights:        service.CloneAPIKeyImageCountWeights(k.ImageCountWeights),
		Quota:                    k.Quota,
		QuotaUsed:                k.QuotaUsed,
		QuotaUsedByCurrency:      cloneUsageCostByCurrency(k.QuotaUsedByCurrency),
		ExpiresAt:                k.ExpiresAt,
		StartsAt:                 k.StartsAt,
		AccessTimePolicy:         k.AccessTimePolicy,
		CreatedAt:                k.CreatedAt,
		UpdatedAt:                k.UpdatedAt,
		RateLimit5h:              k.RateLimit5h,
		RateLimit1d:              k.RateLimit1d,
		RateLimit7d:              k.RateLimit7d,
		Usage5h:                  k.EffectiveUsage5h(),
		Usage1d:                  k.EffectiveUsage1d(),
		Usage7d:                  k.EffectiveUsage7d(),
		Usage5hByCurrency:        cloneUsageCostByCurrency(k.EffectiveUsage5hByCurrency()),
		Usage1dByCurrency:        cloneUsageCostByCurrency(k.EffectiveUsage1dByCurrency()),
		Usage7dByCurrency:        cloneUsageCostByCurrency(k.EffectiveUsage7dByCurrency()),
		Window5hStart:            k.Window5hStart,
		Window1dStart:            k.Window1dStart,
		Window7dStart:            k.Window7dStart,
		User:                     UserFromServiceShallow(k.User),
		Group:                    GroupFromServiceShallow(k.Group),
	}
	if len(k.GroupBindings) > 0 {
		out.GroupIDs = make([]int64, 0, len(k.GroupBindings))
		out.Groups = make([]APIKeyGroupDTO, 0, len(k.GroupBindings))
		for _, binding := range k.GroupBindings {
			out.GroupIDs = append(out.GroupIDs, binding.GroupID)
			dtoBinding := APIKeyGroupDTO{
				GroupID:             binding.GroupID,
				Quota:               binding.Quota,
				QuotaUsed:           binding.QuotaUsed,
				QuotaUsedByCurrency: cloneUsageCostByCurrency(binding.QuotaUsedByCurrency),
				ModelPatterns:       append([]string(nil), binding.ModelPatterns...),
			}
			if binding.Group != nil {
				dtoBinding.GroupName = binding.Group.Name
				dtoBinding.Platform = service.CanonicalizePlatformValue(binding.Group.Platform)
				dtoBinding.Priority = binding.Group.Priority
			}
			out.Groups = append(out.Groups, dtoBinding)
		}
	}
	if k.Window5hStart != nil && !service.IsWindowExpired(k.Window5hStart, service.RateLimitWindow5h) {
		t := k.Window5hStart.Add(service.RateLimitWindow5h)
		out.Reset5hAt = &t
	}
	if k.Window1dStart != nil && !service.IsWindowExpired(k.Window1dStart, service.RateLimitWindow1d) {
		t := k.Window1dStart.Add(service.RateLimitWindow1d)
		out.Reset1dAt = &t
	}
	if k.Window7dStart != nil && !service.IsWindowExpired(k.Window7dStart, service.RateLimitWindow7d) {
		t := k.Window7dStart.Add(service.RateLimitWindow7d)
		out.Reset7dAt = &t
	}
	return out
}
