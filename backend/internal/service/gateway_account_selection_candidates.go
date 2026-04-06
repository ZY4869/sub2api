package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"log/slog"
)

func (s *GatewayService) listSchedulableAccounts(ctx context.Context, groupID *int64, platform string, hasForcePlatform bool) ([]Account, bool, error) {
	if s.schedulerSnapshot != nil {
		accounts, useMixed, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, hasForcePlatform)
		if err == nil {
			accounts = filterGeminiAccountsByPublicProtocol(ctx, accounts, platform)
			slog.Debug("account_scheduling_list_snapshot", "group_id", derefGroupID(groupID), "platform", platform, "use_mixed", useMixed, "count", len(accounts))
			for _, acc := range accounts {
				slog.Debug("account_scheduling_account_detail", "account_id", acc.ID, "name", acc.Name, "platform", acc.Platform, "type", acc.Type, "status", acc.Status, "tls_fingerprint", acc.IsTLSFingerprintEnabled())
			}
		}
		return accounts, useMixed, err
	}
	useMixed := (platform == PlatformAnthropic || platform == PlatformGemini) && !hasForcePlatform
	if useMixed {
		platforms := QueryPlatformsForGroupPlatform(platform, true)
		var accounts []Account
		var err error
		if groupID != nil {
			accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, platforms)
		} else if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
			accounts, err = s.accountRepo.ListSchedulableByPlatforms(ctx, platforms)
		} else {
			accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, platforms)
		}
		if err != nil {
			slog.Debug("account_scheduling_list_failed", "group_id", derefGroupID(groupID), "platform", platform, "error", err)
			return nil, useMixed, err
		}
		filtered := make([]Account, 0, len(accounts))
		for _, acc := range accounts {
			if acc.Platform == PlatformAntigravity && !acc.IsMixedSchedulingEnabled() {
				continue
			}
			if acc.Platform != PlatformAntigravity && !MatchesGroupPlatform(&acc, platform) {
				continue
			}
			filtered = append(filtered, acc)
		}
		filtered = filterGeminiAccountsByPublicProtocol(ctx, filtered, platform)
		slog.Debug("account_scheduling_list_mixed", "group_id", derefGroupID(groupID), "platform", platform, "raw_count", len(accounts), "filtered_count", len(filtered))
		for _, acc := range filtered {
			slog.Debug("account_scheduling_account_detail", "account_id", acc.ID, "name", acc.Name, "platform", acc.Platform, "type", acc.Type, "status", acc.Status, "tls_fingerprint", acc.IsTLSFingerprintEnabled())
		}
		return filtered, useMixed, nil
	}
	var accounts []Account
	var err error
	queryPlatforms := QueryPlatformsForGroupPlatform(platform, false)
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		accounts, err = s.accountRepo.ListSchedulableByPlatforms(ctx, queryPlatforms)
	} else if groupID != nil {
		accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatforms(ctx, *groupID, queryPlatforms)
	} else {
		accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatforms(ctx, queryPlatforms)
	}
	if err != nil {
		slog.Debug("account_scheduling_list_failed", "group_id", derefGroupID(groupID), "platform", platform, "error", err)
		return nil, useMixed, err
	}
	filtered := make([]Account, 0, len(accounts))
	for _, acc := range accounts {
		if MatchesGroupPlatform(&acc, platform) {
			filtered = append(filtered, acc)
		}
	}
	filtered = filterGeminiAccountsByPublicProtocol(ctx, filtered, platform)
	slog.Debug("account_scheduling_list_single", "group_id", derefGroupID(groupID), "platform", platform, "count", len(filtered))
	for _, acc := range filtered {
		slog.Debug("account_scheduling_account_detail", "account_id", acc.ID, "name", acc.Name, "platform", acc.Platform, "type", acc.Type, "status", acc.Status, "tls_fingerprint", acc.IsTLSFingerprintEnabled())
	}
	return filtered, useMixed, nil
}
func (s *GatewayService) IsSingleAntigravityAccountGroup(ctx context.Context, groupID *int64) bool {
	accounts, _, err := s.listSchedulableAccounts(ctx, groupID, PlatformAntigravity, true)
	if err != nil {
		return false
	}
	return len(accounts) == 1
}
func (s *GatewayService) isAccountAllowedForPlatform(account *Account, platform string, useMixed bool) bool {
	if account == nil {
		return false
	}
	if useMixed {
		if MatchesGroupPlatform(account, platform) {
			return true
		}
		return account.Platform == PlatformAntigravity && account.IsMixedSchedulingEnabled()
	}
	return MatchesGroupPlatform(account, platform)
}

func (s *GatewayService) isAccountAllowedForPlatformWithContext(ctx context.Context, account *Account, platform string, useMixed bool) bool {
	if !s.isAccountAllowedForPlatform(account, platform, useMixed) {
		return false
	}
	if platform == PlatformGemini {
		return geminiPublicProtocolAllowsAccount(ctx, account)
	}
	return true
}
func (s *GatewayService) isAccountSchedulableForSelection(account *Account) bool {
	if account == nil {
		return false
	}
	return account.IsSchedulable()
}
func (s *GatewayService) isAccountSchedulableForModelSelection(ctx context.Context, account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	return account.IsSchedulableForModelWithContext(ctx, requestedModel)
}
func (s *GatewayService) isAccountInGroup(account *Account, groupID *int64) bool {
	if account == nil {
		return false
	}
	if groupID == nil {
		return len(account.AccountGroups) == 0
	}
	for _, ag := range account.AccountGroups {
		if ag.GroupID == *groupID {
			return true
		}
	}
	return false
}
func (s *GatewayService) tryAcquireAccountSlot(ctx context.Context, accountID int64, maxConcurrency int) (*AcquireResult, error) {
	if s.concurrencyService == nil {
		return &AcquireResult{Acquired: true, ReleaseFunc: func() {
		}}, nil
	}
	return s.concurrencyService.AcquireAccountSlot(ctx, accountID, maxConcurrency)
}
