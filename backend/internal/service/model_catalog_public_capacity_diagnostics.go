package service

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
	"go.uber.org/zap"
)

const (
	publicCatalogCapacityAvailable     = "available"
	publicCatalogCapacityLimited       = "limited"
	publicCatalogCapacityUnschedulable = "unschedulable"
	publicCatalogCapacityUnknown       = "unknown"
)

const (
	publicCatalogCapacityScopeAccount       = "account"
	publicCatalogCapacityScopeModel         = "model"
	publicCatalogCapacityScopeAPIKey        = "api_key"
	publicCatalogCapacityScopeGroup         = "group"
	publicCatalogCapacityScopeUserPlatform  = "user_platform"
	publicCatalogCapacityScopeProviderQuota = "provider_quota"
)

func (s *ModelCatalogService) PublicModelCatalogCapacityDiagnostics(ctx context.Context) (*PublicModelCatalogCapacityDiagnosticsSnapshot, error) {
	published := s.loadPublishedPublicModelCatalogSnapshot(ctx)
	if published == nil || len(published.Snapshot.Items) == 0 {
		return &PublicModelCatalogCapacityDiagnosticsSnapshot{
			UpdatedAt: time.Now().UTC().Format(time.RFC3339),
			Items:     []PublicModelCatalogCapacityDiagnosticItem{},
			Summary:   PublicModelCatalogCapacityDiagnosticsSummary{},
		}, nil
	}
	published = filterPublicModelCatalogPublishedSnapshotByDemoMode(published, false)
	items := make([]PublicModelCatalogCapacityDiagnosticItem, 0, len(published.Snapshot.Items))
	for _, item := range published.Snapshot.Items {
		items = append(items, s.publicModelCatalogCapacityDiagnosticItem(ctx, item))
	}
	sort.SliceStable(items, func(i, j int) bool {
		return items[i].PublicModelID < items[j].PublicModelID
	})
	summary := summarizePublicModelCatalogCapacityDiagnostics(items)
	fields := []zap.Field{
		zap.String("component", "service.model_catalog"),
		zap.Int("model_count", summary.ModelCount),
		zap.Int("limited_count", summary.LimitedCount),
		zap.Int("unschedulable_count", summary.UnschedulableCount),
		zap.Any("restriction_counts", summary.RestrictionCounts),
	}
	if requestID, _ := ctx.Value(ctxkey.RequestID).(string); strings.TrimSpace(requestID) != "" {
		fields = append(fields, zap.String("request_id", strings.TrimSpace(requestID)))
	}
	logger.FromContext(ctx).Info("public model catalog capacity diagnostics built", fields...)
	return &PublicModelCatalogCapacityDiagnosticsSnapshot{
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
		Items:     items,
		Summary:   summary,
	}, nil
}

func (s *ModelCatalogService) publicModelCatalogCapacityDiagnosticItem(
	ctx context.Context,
	item PublicModelCatalogItem,
) PublicModelCatalogCapacityDiagnosticItem {
	publicID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
	diagnostic := PublicModelCatalogCapacityDiagnosticItem{
		PublicModelID:   publicID,
		Model:           publicID,
		EntryID:         strings.TrimSpace(item.EntryID),
		Provider:        strings.TrimSpace(item.Provider),
		SourceProtocol:  strings.TrimSpace(item.SourceProtocol),
		SourceAccountID: item.SourceAccountID,
		Availability:    publicCatalogCapacityUnknown,
		Sources: []PublicModelCatalogCapacityDiagnosticSource{{
			Source: "published_catalog",
			Scope:  publicCatalogCapacityScopeModel,
			Detail: "published entry snapshot",
		}},
	}
	if s == nil || s.gatewayService == nil || s.gatewayService.accountRepo == nil || item.SourceAccountID <= 0 {
		diagnostic.Restrictions = append(diagnostic.Restrictions, PublicModelCatalogCapacityRestriction{Kind: "source_account_missing", Scope: "account"})
		diagnostic.Availability = publicCatalogCapacityUnschedulable
		return diagnostic
	}
	account, err := s.gatewayService.accountRepo.GetByID(ctx, item.SourceAccountID)
	if err != nil || account == nil {
		diagnostic.Restrictions = append(diagnostic.Restrictions, PublicModelCatalogCapacityRestriction{Kind: "source_account_unavailable", Scope: "account"})
		diagnostic.Availability = publicCatalogCapacityUnschedulable
		return diagnostic
	}
	diagnostic.Sources = append(diagnostic.Sources, PublicModelCatalogCapacityDiagnosticSource{
		Source: "account_pool",
		Scope:  publicCatalogCapacityScopeAccount,
		Detail: "source account scheduling state",
	})
	diagnostic.Restrictions = append(diagnostic.Restrictions, publicModelCatalogAccountRestrictions(ctx, account, item)...)
	if rateLimit := s.publicModelCatalogRateLimitFromAccount(ctx, account, item); publicModelCatalogRateLimitSummaryHasValue(rateLimit) {
		diagnostic.EffectiveRateLimit = clonePublicModelCatalogRateLimitSummary(rateLimit)
		diagnostic.Sources = append(diagnostic.Sources, PublicModelCatalogCapacityDiagnosticSource{
			Source: "provider_quota",
			Scope:  publicCatalogCapacityScopeProviderQuota,
			Detail: "provider quota or RPM summary",
		})
	}
	groupIDs := publicModelCatalogAccountGroupIDs(account)
	if len(groupIDs) > 0 {
		diagnostic.BindingGroupID = groupIDs[0]
	}
	diagnostic.Sources = append(diagnostic.Sources, s.publicModelCatalogGroupDiagnosticSources(ctx, groupIDs)...)
	diagnostic.Restrictions = append(diagnostic.Restrictions, s.publicModelCatalogGroupRestrictions(ctx, groupIDs)...)
	diagnostic.Sources = append(diagnostic.Sources, s.publicModelCatalogAPIKeyDiagnosticSources(ctx, groupIDs)...)
	diagnostic.Restrictions = append(diagnostic.Restrictions, s.publicModelCatalogAPIKeyRestrictions(ctx, groupIDs, account)...)
	diagnostic.Availability = publicModelCatalogCapacityAvailability(diagnostic.Restrictions)
	return diagnostic
}

func publicModelCatalogAccountRestrictions(ctx context.Context, account *Account, item PublicModelCatalogItem) []PublicModelCatalogCapacityRestriction {
	now := time.Now().UTC()
	restrictions := []PublicModelCatalogCapacityRestriction{}
	if account == nil {
		return []PublicModelCatalogCapacityRestriction{{Kind: "source_account_missing", Scope: publicCatalogCapacityScopeAccount}}
	}
	if !account.IsActive() {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "account_inactive", Scope: publicCatalogCapacityScopeAccount})
	}
	if !account.Schedulable {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "account_unschedulable", Scope: publicCatalogCapacityScopeAccount})
	}
	if account.OverloadUntil != nil && now.Before(account.OverloadUntil.UTC()) {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "account_overloaded", Scope: publicCatalogCapacityScopeAccount, Until: account.OverloadUntil.UTC().Format(time.RFC3339)})
	}
	if account.RateLimitResetAt != nil && now.Before(account.RateLimitResetAt.UTC()) {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "account_rate_limited", Scope: publicCatalogCapacityScopeAccount, Until: account.RateLimitResetAt.UTC().Format(time.RFC3339)})
	}
	if account.TempUnschedulableUntil != nil && now.Before(account.TempUnschedulableUntil.UTC()) {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "account_cooling_down", Scope: publicCatalogCapacityScopeAccount, Until: account.TempUnschedulableUntil.UTC().Format(time.RFC3339)})
	}
	if account.AutoPauseOnExpired && account.ExpiresAt != nil && !now.Before(account.ExpiresAt.UTC()) {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "account_expired", Scope: publicCatalogCapacityScopeAccount})
	}
	if account.IsQuotaExceeded() {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "account_quota_exhausted", Scope: publicCatalogCapacityScopeAccount})
	}
	restrictions = append(restrictions, publicModelCatalogAccountQuotaRestrictions(account)...)
	if resetAt := account.GetModelRateLimitRemainingTimeWithContext(ctx, firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel, item.Model)); resetAt > 0 {
		until := time.Now().UTC().Add(resetAt).Format(time.RFC3339)
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "model_rate_limited", Scope: publicCatalogCapacityScopeModel, Until: until})
	}
	if account.IsSchedulable() && !account.IsSchedulableForModelWithContext(ctx, firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel, item.Model)) {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "model_rate_limited", Scope: publicCatalogCapacityScopeModel})
	}
	return restrictions
}

func publicModelCatalogAccountQuotaRestrictions(account *Account) []PublicModelCatalogCapacityRestriction {
	if account == nil {
		return nil
	}
	restrictions := []PublicModelCatalogCapacityRestriction{}
	appendQuota := func(kind string, limit float64, used float64) {
		if limit <= 0 {
			return
		}
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{
			Kind:  kind,
			Scope: publicCatalogCapacityScopeAccount,
			Limit: modelCatalogFloat64Ptr(limit),
			Used:  modelCatalogFloat64Ptr(used),
		})
	}
	appendQuota("account_quota_configured", account.GetQuotaLimit(), account.GetQuotaUsed())
	appendQuota("account_daily_quota_configured", account.GetQuotaDailyLimit(), account.GetQuotaDailyUsed())
	appendQuota("account_weekly_quota_configured", account.GetQuotaWeeklyLimit(), account.GetQuotaWeeklyUsed())
	return restrictions
}

func (s *ModelCatalogService) publicModelCatalogGroupDiagnosticSources(
	ctx context.Context,
	groupIDs []int64,
) []PublicModelCatalogCapacityDiagnosticSource {
	if s == nil || s.gatewayService == nil || s.gatewayService.groupRepo == nil || len(groupIDs) == 0 {
		return nil
	}
	sources := make([]PublicModelCatalogCapacityDiagnosticSource, 0, len(groupIDs))
	for _, groupID := range groupIDs {
		group, _ := s.gatewayService.groupRepo.GetByIDLite(ctx, groupID)
		detail := "group_id=" + strconv.FormatInt(groupID, 10)
		if group != nil && strings.TrimSpace(group.Name) != "" {
			detail += " name=" + strings.TrimSpace(group.Name)
		}
		sources = append(sources, PublicModelCatalogCapacityDiagnosticSource{
			Source: "group_quota",
			Scope:  publicCatalogCapacityScopeGroup,
			Detail: detail,
		})
	}
	return sources
}

func (s *ModelCatalogService) publicModelCatalogGroupRestrictions(
	ctx context.Context,
	groupIDs []int64,
) []PublicModelCatalogCapacityRestriction {
	if s == nil || s.gatewayService == nil || s.gatewayService.groupRepo == nil || len(groupIDs) == 0 {
		return nil
	}
	restrictions := []PublicModelCatalogCapacityRestriction{}
	for _, groupID := range groupIDs {
		group, err := s.gatewayService.groupRepo.GetByIDLite(ctx, groupID)
		if err != nil || group == nil {
			restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "group_unavailable", Scope: publicCatalogCapacityScopeGroup})
			continue
		}
		if !group.IsActive() {
			restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "group_inactive", Scope: publicCatalogCapacityScopeGroup})
		}
		restrictions = append(restrictions, publicModelCatalogGroupQuotaRestrictions(ctx, s.gatewayService.userSubRepo, group)...)
	}
	return restrictions
}

func publicModelCatalogGroupQuotaRestrictions(
	ctx context.Context,
	repo UserSubscriptionRepository,
	group *Group,
) []PublicModelCatalogCapacityRestriction {
	if group == nil || repo == nil || !group.IsSubscriptionType() {
		return nil
	}
	subs, _, err := repo.ListByGroupID(ctx, group.ID, pagination.PaginationParams{Page: 1, PageSize: 100})
	if err != nil {
		return []PublicModelCatalogCapacityRestriction{{Kind: "group_subscription_quota_unavailable", Scope: publicCatalogCapacityScopeGroup}}
	}
	restrictions := []PublicModelCatalogCapacityRestriction{}
	appendQuota := func(kind string, limit *float64, used float64, resetAt *time.Time) {
		if limit == nil || *limit <= 0 {
			return
		}
		restriction := PublicModelCatalogCapacityRestriction{
			Kind:  kind,
			Scope: publicCatalogCapacityScopeGroup,
			Limit: cloneFloat64Ptr(limit),
			Used:  modelCatalogFloat64Ptr(used),
		}
		if resetAt != nil {
			restriction.Until = resetAt.UTC().Format(time.RFC3339)
		}
		restrictions = append(restrictions, restriction)
	}
	for _, sub := range subs {
		if sub.Status != SubscriptionStatusActive {
			continue
		}
		appendQuota("group_daily_quota_configured", group.DailyLimitUSD, sub.DailyUsageUSD, sub.DailyResetTime())
		appendQuota("group_weekly_quota_configured", group.WeeklyLimitUSD, sub.WeeklyUsageUSD, sub.WeeklyResetTime())
		appendQuota("group_monthly_quota_configured", group.MonthlyLimitUSD, sub.MonthlyUsageUSD, sub.MonthlyResetTime())
		if !sub.CheckDailyLimit(group, 0) {
			restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "group_daily_quota_exhausted", Scope: publicCatalogCapacityScopeGroup})
		}
		if !sub.CheckWeeklyLimit(group, 0) {
			restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "group_weekly_quota_exhausted", Scope: publicCatalogCapacityScopeGroup})
		}
		if !sub.CheckMonthlyLimit(group, 0) {
			restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "group_monthly_quota_exhausted", Scope: publicCatalogCapacityScopeGroup})
		}
	}
	return restrictions
}

func (s *ModelCatalogService) publicModelCatalogAPIKeyDiagnosticSources(
	ctx context.Context,
	groupIDs []int64,
) []PublicModelCatalogCapacityDiagnosticSource {
	apiKeys := s.publicModelCatalogAPIKeysForGroups(ctx, groupIDs)
	if len(apiKeys) == 0 {
		return nil
	}
	sources := make([]PublicModelCatalogCapacityDiagnosticSource, 0, len(apiKeys))
	for _, key := range apiKeys {
		detail := "api_key_id=" + strconv.FormatInt(key.ID, 10)
		if key.UserID > 0 {
			detail += " user_id=" + strconv.FormatInt(key.UserID, 10)
		}
		sources = append(sources, PublicModelCatalogCapacityDiagnosticSource{
			Source: "api_key_limit",
			Scope:  publicCatalogCapacityScopeAPIKey,
			Detail: detail,
		})
	}
	if s != nil && s.userPlatformQuotas != nil {
		for _, userID := range publicModelCatalogAPIKeyUserIDs(apiKeys) {
			sources = append(sources, PublicModelCatalogCapacityDiagnosticSource{
				Source: "user_platform_quota",
				Scope:  publicCatalogCapacityScopeUserPlatform,
				Detail: "user_id=" + strconv.FormatInt(userID, 10),
			})
		}
	}
	return sources
}

func (s *ModelCatalogService) publicModelCatalogAPIKeyRestrictions(
	ctx context.Context,
	groupIDs []int64,
	account *Account,
) []PublicModelCatalogCapacityRestriction {
	apiKeys := s.publicModelCatalogAPIKeysForGroups(ctx, groupIDs)
	if len(apiKeys) == 0 {
		return nil
	}
	restrictions := []PublicModelCatalogCapacityRestriction{}
	seenUsers := map[int64]struct{}{}
	for _, key := range apiKeys {
		if key.UserID > 0 {
			seenUsers[key.UserID] = struct{}{}
		}
		restrictions = append(restrictions, publicModelCatalogAPIKeyQuotaRestrictions(key)...)
		restrictions = append(restrictions, s.publicModelCatalogAPIKeyRateLimitRestrictions(ctx, key)...)
		restrictions = append(restrictions, publicModelCatalogAPIKeyGroupBindingRestrictions(key)...)
	}
	userIDs := make([]int64, 0, len(seenUsers))
	for userID := range seenUsers {
		userIDs = append(userIDs, userID)
	}
	sort.SliceStable(userIDs, func(i, j int) bool { return userIDs[i] < userIDs[j] })
	for _, userID := range userIDs {
		restrictions = append(restrictions, s.publicModelCatalogUserPlatformQuotaRestrictions(ctx, userID, account)...)
	}
	return restrictions
}

func publicModelCatalogAPIKeyUserIDs(apiKeys []APIKey) []int64 {
	seen := map[int64]struct{}{}
	out := []int64{}
	for _, key := range apiKeys {
		if key.UserID <= 0 {
			continue
		}
		if _, ok := seen[key.UserID]; ok {
			continue
		}
		seen[key.UserID] = struct{}{}
		out = append(out, key.UserID)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func (s *ModelCatalogService) publicModelCatalogAPIKeysForGroups(ctx context.Context, groupIDs []int64) []APIKey {
	if s == nil || s.apiKeyRepo == nil || len(groupIDs) == 0 {
		return nil
	}
	seen := map[int64]struct{}{}
	apiKeys := []APIKey{}
	for _, groupID := range groupIDs {
		keys, _, err := s.apiKeyRepo.ListByGroupID(ctx, groupID, pagination.PaginationParams{Page: 1, PageSize: 100})
		if err != nil {
			continue
		}
		for _, key := range keys {
			if key.ID <= 0 {
				continue
			}
			if _, ok := seen[key.ID]; ok {
				continue
			}
			seen[key.ID] = struct{}{}
			apiKeys = append(apiKeys, key)
		}
	}
	sort.SliceStable(apiKeys, func(i, j int) bool { return apiKeys[i].ID < apiKeys[j].ID })
	return apiKeys
}

func publicModelCatalogAPIKeyQuotaRestrictions(key APIKey) []PublicModelCatalogCapacityRestriction {
	restrictions := []PublicModelCatalogCapacityRestriction{}
	if !key.IsActive() {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "api_key_inactive", Scope: publicCatalogCapacityScopeAPIKey})
	}
	if key.IsExpired() {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{Kind: "api_key_expired", Scope: publicCatalogCapacityScopeAPIKey})
	}
	if key.IsQuotaExhausted() {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{
			Kind:  "api_key_quota_exhausted",
			Scope: publicCatalogCapacityScopeAPIKey,
			Limit: modelCatalogFloat64Ptr(key.Quota),
			Used:  modelCatalogFloat64Ptr(key.QuotaUsed),
		})
	} else if key.Quota > 0 {
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{
			Kind:  "api_key_quota_configured",
			Scope: publicCatalogCapacityScopeAPIKey,
			Limit: modelCatalogFloat64Ptr(key.Quota),
			Used:  modelCatalogFloat64Ptr(key.QuotaUsed),
		})
	}
	return restrictions
}

func (s *ModelCatalogService) publicModelCatalogAPIKeyRateLimitRestrictions(
	ctx context.Context,
	key APIKey,
) []PublicModelCatalogCapacityRestriction {
	if !key.HasRateLimits() {
		return nil
	}
	data := &APIKeyRateLimitData{
		Usage5h:       key.Usage5h,
		Usage1d:       key.Usage1d,
		Usage7d:       key.Usage7d,
		Window5hStart: key.Window5hStart,
		Window1dStart: key.Window1dStart,
		Window7dStart: key.Window7dStart,
	}
	if s != nil && s.apiKeyRepo != nil {
		if loaded, err := s.apiKeyRepo.GetRateLimitData(ctx, key.ID); err == nil && loaded != nil {
			data = loaded
		}
	}
	restrictions := []PublicModelCatalogCapacityRestriction{}
	appendWindow := func(kind string, limit float64, used float64, start *time.Time, window time.Duration) {
		if limit <= 0 {
			return
		}
		restrictionKind := kind + "_configured"
		if used >= limit {
			restrictionKind = kind + "_exhausted"
		}
		restriction := PublicModelCatalogCapacityRestriction{
			Kind:  restrictionKind,
			Scope: publicCatalogCapacityScopeAPIKey,
			Limit: modelCatalogFloat64Ptr(limit),
			Used:  modelCatalogFloat64Ptr(used),
		}
		if start != nil {
			restriction.Until = start.UTC().Add(window).Format(time.RFC3339)
		}
		restrictions = append(restrictions, restriction)
	}
	appendWindow("api_key_rate_limit_5h", key.RateLimit5h, data.EffectiveUsage5h(), data.Window5hStart, RateLimitWindow5h)
	appendWindow("api_key_rate_limit_1d", key.RateLimit1d, data.EffectiveUsage1d(), data.Window1dStart, RateLimitWindow1d)
	appendWindow("api_key_rate_limit_7d", key.RateLimit7d, data.EffectiveUsage7d(), data.Window7dStart, RateLimitWindow7d)
	return restrictions
}

func publicModelCatalogAPIKeyGroupBindingRestrictions(key APIKey) []PublicModelCatalogCapacityRestriction {
	restrictions := []PublicModelCatalogCapacityRestriction{}
	for _, binding := range key.GroupBindings {
		if binding.Quota <= 0 {
			continue
		}
		kind := "api_key_group_quota_configured"
		if binding.IsQuotaExhausted() {
			kind = "api_key_group_quota_exhausted"
		}
		restrictions = append(restrictions, PublicModelCatalogCapacityRestriction{
			Kind:  kind,
			Scope: publicCatalogCapacityScopeGroup,
			Limit: modelCatalogFloat64Ptr(binding.Quota),
			Used:  modelCatalogFloat64Ptr(binding.QuotaUsed),
		})
	}
	return restrictions
}

func (s *ModelCatalogService) publicModelCatalogUserPlatformQuotaRestrictions(
	ctx context.Context,
	userID int64,
	account *Account,
) []PublicModelCatalogCapacityRestriction {
	if s == nil || s.userPlatformQuotas == nil || userID <= 0 {
		return nil
	}
	platform := UserPlatformQuotaPlatformForAccount(account)
	views, err := s.userPlatformQuotas.ListUserQuotas(ctx, userID)
	if err != nil {
		return []PublicModelCatalogCapacityRestriction{{Kind: "user_platform_quota_unavailable", Scope: publicCatalogCapacityScopeUserPlatform}}
	}
	restrictions := []PublicModelCatalogCapacityRestriction{}
	for _, view := range views {
		if platform != "" && NormalizeUserPlatformQuotaPlatform(view.Platform) != platform {
			continue
		}
		restrictions = append(restrictions, publicModelCatalogUserPlatformQuotaCycleRestriction("user_platform_daily_quota", view.Daily)...)
		restrictions = append(restrictions, publicModelCatalogUserPlatformQuotaCycleRestriction("user_platform_weekly_quota", view.Weekly)...)
		restrictions = append(restrictions, publicModelCatalogUserPlatformQuotaCycleRestriction("user_platform_monthly_quota", view.Monthly)...)
	}
	return restrictions
}

func publicModelCatalogUserPlatformQuotaCycleRestriction(
	kind string,
	cycle UserPlatformQuotaCycle,
) []PublicModelCatalogCapacityRestriction {
	if cycle.LimitUSD == nil || *cycle.LimitUSD <= 0 {
		return nil
	}
	restrictionKind := kind + "_configured"
	if cycle.UsageUSD >= *cycle.LimitUSD {
		restrictionKind = kind + "_exhausted"
	}
	restriction := PublicModelCatalogCapacityRestriction{
		Kind:  restrictionKind,
		Scope: publicCatalogCapacityScopeUserPlatform,
		Limit: cloneFloat64Ptr(cycle.LimitUSD),
		Used:  modelCatalogFloat64Ptr(cycle.UsageUSD),
	}
	if cycle.ResetAt != nil {
		restriction.Until = cycle.ResetAt.UTC().Format(time.RFC3339)
	}
	return []PublicModelCatalogCapacityRestriction{restriction}
}

func publicModelCatalogAccountGroupIDs(account *Account) []int64 {
	if account == nil {
		return nil
	}
	seen := map[int64]struct{}{}
	out := []int64{}
	appendID := func(id int64) {
		if id <= 0 {
			return
		}
		if _, ok := seen[id]; ok {
			return
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	for _, id := range account.GroupIDs {
		appendID(id)
	}
	for _, binding := range account.AccountGroups {
		appendID(binding.GroupID)
	}
	for _, group := range account.Groups {
		if group != nil {
			appendID(group.ID)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

func publicModelCatalogCapacityAvailability(restrictions []PublicModelCatalogCapacityRestriction) string {
	if len(restrictions) == 0 {
		return publicCatalogCapacityAvailable
	}
	for _, restriction := range restrictions {
		if strings.HasSuffix(strings.TrimSpace(restriction.Kind), "_configured") {
			continue
		}
		switch restriction.Kind {
		case "account_rate_limited",
			"account_cooling_down",
			"model_rate_limited",
			"account_overloaded",
			"api_key_rate_limit_5h_exhausted",
			"api_key_rate_limit_1d_exhausted",
			"api_key_rate_limit_7d_exhausted":
			return publicCatalogCapacityLimited
		}
	}
	return publicCatalogCapacityUnschedulable
}

func summarizePublicModelCatalogCapacityDiagnostics(items []PublicModelCatalogCapacityDiagnosticItem) PublicModelCatalogCapacityDiagnosticsSummary {
	summary := PublicModelCatalogCapacityDiagnosticsSummary{
		ModelCount:           len(items),
		SourceCounts:         map[string]int{},
		RestrictionCounts:    map[string]int{},
		EffectiveLimitCounts: map[string]int{},
	}
	for _, item := range items {
		switch item.Availability {
		case publicCatalogCapacityAvailable:
			summary.AvailableCount++
		case publicCatalogCapacityLimited:
			summary.LimitedCount++
		case publicCatalogCapacityUnschedulable:
			summary.UnschedulableCount++
		}
		for _, source := range item.Sources {
			if strings.TrimSpace(source.Source) != "" {
				summary.SourceCounts[source.Source]++
			}
		}
		for _, restriction := range item.Restrictions {
			if strings.TrimSpace(restriction.Kind) != "" {
				summary.RestrictionCounts[restriction.Kind]++
			}
		}
		if item.EffectiveRateLimit != nil {
			if item.EffectiveRateLimit.RPM != nil {
				summary.EffectiveLimitCounts["rpm"]++
			}
			if item.EffectiveRateLimit.TPM != nil {
				summary.EffectiveLimitCounts["tpm"]++
			}
			if item.EffectiveRateLimit.RPD != nil {
				summary.EffectiveLimitCounts["rpd"]++
			}
		}
	}
	if len(summary.SourceCounts) == 0 {
		summary.SourceCounts = nil
	}
	if len(summary.RestrictionCounts) == 0 {
		summary.RestrictionCounts = nil
	}
	if len(summary.EffectiveLimitCounts) == 0 {
		summary.EffectiveLimitCounts = nil
	}
	return summary
}
