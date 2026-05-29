package service

import "context"

func (s *OpsAlertEvaluatorService) computeOpsAlertAvailabilityMetric(
	ctx context.Context,
	metricType string,
	platform string,
	groupID *int64,
	evalCache *opsAlertEvaluationCache,
) (float64, bool) {
	switch metricType {
	case "group_available_accounts":
		if groupID == nil || *groupID <= 0 || s == nil || s.opsService == nil {
			return 0, false
		}
		availability, err := s.getCachedAccountAvailability(ctx, platform, groupID, evalCache)
		if err != nil || availability == nil {
			return 0, false
		}
		if availability.Group == nil {
			return 0, true
		}
		return float64(availability.Group.AvailableCount), true
	case "group_available_ratio":
		if groupID == nil || *groupID <= 0 || s == nil || s.opsService == nil {
			return 0, false
		}
		availability, err := s.getCachedAccountAvailability(ctx, platform, groupID, evalCache)
		if err != nil || availability == nil {
			return 0, false
		}
		return computeGroupAvailableRatio(availability.Group), true
	case "account_rate_limited_count":
		return s.countOpsAlertAvailabilityAccounts(ctx, platform, groupID, evalCache, func(acc *AccountAvailability) bool {
			return acc.IsRateLimited
		})
	case "account_error_count":
		return s.countOpsAlertAvailabilityAccounts(ctx, platform, groupID, evalCache, func(acc *AccountAvailability) bool {
			return acc.HasError && acc.TempUnschedulableUntil == nil
		})
	case "group_rate_limit_ratio":
		if groupID == nil || *groupID <= 0 || s == nil || s.opsService == nil {
			return 0, false
		}
		availability, err := s.getCachedAccountAvailability(ctx, platform, groupID, evalCache)
		if err != nil || availability == nil {
			return 0, false
		}
		if availability.Group == nil || availability.Group.TotalAccounts <= 0 {
			return 0, true
		}
		return (float64(availability.Group.RateLimitCount) / float64(availability.Group.TotalAccounts)) * 100, true
	case "account_error_ratio":
		if s == nil || s.opsService == nil {
			return 0, false
		}
		availability, err := s.getCachedAccountAvailability(ctx, platform, groupID, evalCache)
		if err != nil || availability == nil {
			return 0, false
		}
		total := int64(len(availability.Accounts))
		if total <= 0 {
			return 0, true
		}
		errorCount := countAccountsByCondition(availability.Accounts, func(acc *AccountAvailability) bool {
			return acc.HasError && acc.TempUnschedulableUntil == nil
		})
		return (float64(errorCount) / float64(total)) * 100, true
	case "overload_account_count":
		return s.countOpsAlertAvailabilityAccounts(ctx, platform, groupID, evalCache, func(acc *AccountAvailability) bool {
			return acc.IsOverloaded
		})
	default:
		return 0, false
	}
}

func (s *OpsAlertEvaluatorService) countOpsAlertAvailabilityAccounts(
	ctx context.Context,
	platform string,
	groupID *int64,
	evalCache *opsAlertEvaluationCache,
	condition func(*AccountAvailability) bool,
) (float64, bool) {
	if s == nil || s.opsService == nil {
		return 0, false
	}
	availability, err := s.getCachedAccountAvailability(ctx, platform, groupID, evalCache)
	if err != nil || availability == nil {
		return 0, false
	}
	return float64(countAccountsByCondition(availability.Accounts, condition)), true
}
