package dto

import (
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/service"
)

var accountListLiteExtraAllowlist = map[string]struct{}{
	"email_address":     {},
	"privacy_mode":      {},
	"model_rate_limits": {},
	"allow_overages":    {},
}

func AccountFromServiceShallow(a *service.Account) *Account {
	if a == nil {
		return nil
	}
	out := accountFromServiceBase(a, time.Now())
	applyAccountGatewayBatch(a, out)
	applyAccountAnthropicRuntime(a, out)
	applyAccountProtocolGatewayMimic(a, out)
	applyAccountQuota(a, out)
	applyAccountGeminiBatch(a, out)
	return out
}

func accountFromServiceBase(a *service.Account, now time.Time) *Account {
	displayRateLimit := service.AccountDisplayRateLimitState(a, now)
	credentials := RedactAccountCredentials(a.Credentials)
	if enriched, changed := service.EnrichOpenAIOAuthCredentials(a.Platform, a.Type, a.Credentials); changed {
		credentials = RedactAccountCredentials(enriched)
	}
	var rateLimitedAt *time.Time
	var rateLimitResetAt *time.Time
	rateLimitReason := ""
	if displayRateLimit.Limited {
		rateLimitResetAt = displayRateLimit.ResetAt
		if strings.TrimSpace(displayRateLimit.Reason) != "" {
			rateLimitReason = displayRateLimit.Reason
		}
		rateLimitedAt = a.RateLimitedAt
		if rateLimitedAt == nil {
			projected := now.UTC()
			rateLimitedAt = &projected
		}
	}
	return &Account{
		ID:                      a.ID,
		Name:                    a.Name,
		Notes:                   a.Notes,
		Platform:                service.CanonicalizePlatformValue(a.Platform),
		GatewayProtocol:         service.GetAccountGatewayProtocol(a),
		Type:                    a.Type,
		ActiveUsageAvailable:    isAccountActiveUsageAvailable(a),
		Credentials:             credentials,
		Extra:                   a.Extra,
		ProxyID:                 a.ProxyID,
		OriginalProxyID:         a.OriginalProxyID,
		OriginalProxyName:       a.OriginalProxyName,
		Concurrency:             a.Concurrency,
		LoadFactor:              a.LoadFactor,
		Priority:                a.Priority,
		RateMultiplier:          a.BillingRateMultiplier(),
		Status:                  service.PresentAdminAccountStatus(a.Status),
		LifecycleState:          service.NormalizeAccountLifecycleInput(a.LifecycleState),
		LifecycleReasonCode:     a.LifecycleReasonCode,
		LifecycleReasonMessage:  a.LifecycleReasonMessage,
		ErrorMessage:            a.ErrorMessage,
		LastUsedAt:              a.LastUsedAt,
		ExpiresAt:               timeToUnixSeconds(a.ExpiresAt),
		AutoPauseOnExpired:      a.AutoPauseOnExpired,
		AutoRenewEnabled:        a.AutoRenewEnabled,
		AutoRenewPeriod:         a.AutoRenewPeriod,
		CreatedAt:               a.CreatedAt,
		UpdatedAt:               a.UpdatedAt,
		BlacklistedAt:           a.BlacklistedAt,
		BlacklistPurgeAt:        a.BlacklistPurgeAt,
		Schedulable:             a.Schedulable,
		RateLimitedAt:           rateLimitedAt,
		RateLimitResetAt:        rateLimitResetAt,
		RateLimitReason:         rateLimitReason,
		OverloadUntil:           a.OverloadUntil,
		TempUnschedulableUntil:  a.TempUnschedulableUntil,
		TempUnschedulableReason: a.TempUnschedulableReason,
		AutoRecoveryProbe:       accountAutoRecoveryProbeSummaryFromService(service.AccountAutoRecoveryProbeSummaryFromExtra(a.Extra)),
		SessionWindowStart:      a.SessionWindowStart,
		SessionWindowEnd:        a.SessionWindowEnd,
		SessionWindowStatus:     a.SessionWindowStatus,
		GroupIDs:                a.GroupIDs,
	}
}

func applyAccountGatewayBatch(a *service.Account, out *Account) {
	if service.IsProtocolGatewayAccount(a) {
		gatewayBatchEnabled := a.IsGatewayBatchEnabled()
		out.GatewayBatchEnabled = &gatewayBatchEnabled
	}
}

func AccountFromService(a *service.Account) *Account {
	if a == nil {
		return nil
	}
	out := AccountFromServiceShallow(a)
	out.Proxy = ProxyFromService(a.Proxy)
	if len(a.AccountGroups) > 0 {
		out.AccountGroups = make([]AccountGroup, 0, len(a.AccountGroups))
		for i := range a.AccountGroups {
			ag := a.AccountGroups[i]
			out.AccountGroups = append(out.AccountGroups, *AccountGroupFromService(&ag))
		}
	}
	if len(a.Groups) > 0 {
		out.Groups = make([]*Group, 0, len(a.Groups))
		for _, g := range a.Groups {
			out.Groups = append(out.Groups, GroupFromServiceShallow(g))
		}
	}
	return out
}

func AccountFromServiceListLite(a *service.Account) *Account {
	if a == nil {
		return nil
	}
	out := AccountFromService(a)
	out.Credentials = filterAccountListLiteMap(out.Credentials, accountListLiteCredentialAllowlist, false)
	out.Extra = filterAccountListLiteMap(out.Extra, accountListLiteExtraAllowlist, true)
	return out
}

func filterAccountListLiteMap(source map[string]any, allowlist map[string]struct{}, includeCodexPrefix bool) map[string]any {
	if source == nil {
		return nil
	}
	filtered := make(map[string]any)
	for key, value := range source {
		if _, ok := allowlist[key]; ok {
			filtered[key] = value
			continue
		}
		if includeCodexPrefix && strings.HasPrefix(key, "codex_") {
			filtered[key] = value
		}
	}
	if len(filtered) == 0 {
		return map[string]any{}
	}
	return filtered
}

func accountAutoRecoveryProbeSummaryFromService(summary *service.AccountAutoRecoveryProbeSummary) *AccountAutoRecoveryProbeSummary {
	if summary == nil {
		return nil
	}
	return &AccountAutoRecoveryProbeSummary{
		CheckedAt:   summary.CheckedAt,
		Status:      summary.Status,
		Summary:     summary.Summary,
		Blacklisted: summary.Blacklisted,
		NextRetryAt: summary.NextRetryAt,
		ErrorCode:   summary.ErrorCode,
	}
}
