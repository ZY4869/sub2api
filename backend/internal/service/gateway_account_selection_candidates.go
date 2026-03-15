package service

import (
	"context"
	"fmt"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"log/slog"
	"time"
)

func (s *GatewayService) listSchedulableAccounts(ctx context.Context, groupID *int64, platform string, hasForcePlatform bool) ([]Account, bool, error) {
	if platform == PlatformSora {
		return s.listSoraSchedulableAccounts(ctx, groupID)
	}
	if s.schedulerSnapshot != nil {
		accounts, useMixed, err := s.schedulerSnapshot.ListSchedulableAccounts(ctx, groupID, platform, hasForcePlatform)
		if err == nil {
			slog.Debug("account_scheduling_list_snapshot", "group_id", derefGroupID(groupID), "platform", platform, "use_mixed", useMixed, "count", len(accounts))
			for _, acc := range accounts {
				slog.Debug("account_scheduling_account_detail", "account_id", acc.ID, "name", acc.Name, "platform", acc.Platform, "type", acc.Type, "status", acc.Status, "tls_fingerprint", acc.IsTLSFingerprintEnabled())
			}
		}
		return accounts, useMixed, err
	}
	useMixed := (platform == PlatformAnthropic || platform == PlatformGemini) && !hasForcePlatform
	if useMixed {
		platforms := []string{platform, PlatformAntigravity}
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
			filtered = append(filtered, acc)
		}
		slog.Debug("account_scheduling_list_mixed", "group_id", derefGroupID(groupID), "platform", platform, "raw_count", len(accounts), "filtered_count", len(filtered))
		for _, acc := range filtered {
			slog.Debug("account_scheduling_account_detail", "account_id", acc.ID, "name", acc.Name, "platform", acc.Platform, "type", acc.Type, "status", acc.Status, "tls_fingerprint", acc.IsTLSFingerprintEnabled())
		}
		return filtered, useMixed, nil
	}
	var accounts []Account
	var err error
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		accounts, err = s.accountRepo.ListSchedulableByPlatform(ctx, platform)
	} else if groupID != nil {
		accounts, err = s.accountRepo.ListSchedulableByGroupIDAndPlatform(ctx, *groupID, platform)
	} else {
		accounts, err = s.accountRepo.ListSchedulableUngroupedByPlatform(ctx, platform)
	}
	if err != nil {
		slog.Debug("account_scheduling_list_failed", "group_id", derefGroupID(groupID), "platform", platform, "error", err)
		return nil, useMixed, err
	}
	slog.Debug("account_scheduling_list_single", "group_id", derefGroupID(groupID), "platform", platform, "count", len(accounts))
	for _, acc := range accounts {
		slog.Debug("account_scheduling_account_detail", "account_id", acc.ID, "name", acc.Name, "platform", acc.Platform, "type", acc.Type, "status", acc.Status, "tls_fingerprint", acc.IsTLSFingerprintEnabled())
	}
	return accounts, useMixed, nil
}
func (s *GatewayService) listSoraSchedulableAccounts(ctx context.Context, groupID *int64) ([]Account, bool, error) {
	const useMixed = false
	var accounts []Account
	var err error
	if s.cfg != nil && s.cfg.RunMode == config.RunModeSimple {
		accounts, err = s.accountRepo.ListByPlatform(ctx, PlatformSora)
	} else if groupID != nil {
		accounts, err = s.accountRepo.ListByGroup(ctx, *groupID)
	} else {
		accounts, err = s.accountRepo.ListByPlatform(ctx, PlatformSora)
	}
	if err != nil {
		slog.Debug("account_scheduling_list_failed", "group_id", derefGroupID(groupID), "platform", PlatformSora, "error", err)
		return nil, useMixed, err
	}
	filtered := make([]Account, 0, len(accounts))
	for _, acc := range accounts {
		if acc.Platform != PlatformSora {
			continue
		}
		if !s.isSoraAccountSchedulable(&acc) {
			continue
		}
		filtered = append(filtered, acc)
	}
	slog.Debug("account_scheduling_list_sora", "group_id", derefGroupID(groupID), "platform", PlatformSora, "raw_count", len(accounts), "filtered_count", len(filtered))
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
		if account.Platform == platform {
			return true
		}
		return account.Platform == PlatformAntigravity && account.IsMixedSchedulingEnabled()
	}
	return account.Platform == platform
}
func (s *GatewayService) isSoraAccountSchedulable(account *Account) bool {
	return s.soraUnschedulableReason(account) == ""
}
func (s *GatewayService) soraUnschedulableReason(account *Account) string {
	if account == nil {
		return "account_nil"
	}
	if account.Status != StatusActive {
		return fmt.Sprintf("status=%s", account.Status)
	}
	if !account.Schedulable {
		return "schedulable=false"
	}
	if account.TempUnschedulableUntil != nil && time.Now().Before(*account.TempUnschedulableUntil) {
		return fmt.Sprintf("temp_unschedulable_until=%s", account.TempUnschedulableUntil.UTC().Format(time.RFC3339))
	}
	return ""
}
func (s *GatewayService) isAccountSchedulableForSelection(account *Account) bool {
	if account == nil {
		return false
	}
	if account.Platform == PlatformSora {
		return s.isSoraAccountSchedulable(account)
	}
	return account.IsSchedulable()
}
func (s *GatewayService) isAccountSchedulableForModelSelection(ctx context.Context, account *Account, requestedModel string) bool {
	if account == nil {
		return false
	}
	if account.Platform == PlatformSora {
		if !s.isSoraAccountSchedulable(account) {
			return false
		}
		return account.GetRateLimitRemainingTimeWithContext(ctx, requestedModel) <= 0
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
