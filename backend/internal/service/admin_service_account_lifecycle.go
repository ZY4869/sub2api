package service

import (
	"context"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const blacklistedRestoreForbiddenMessage = "blacklisted accounts can only be restored via blacklist retest"
const (
	manualBlacklistReasonCode    = "manual_blacklist"
	manualBlacklistReasonMessage = "Added to blacklist by admin"
)

func normalizeAccountLifecycleWriteInput(raw string) string {
	lifecycle := NormalizeAccountLifecycleInput(raw)
	if lifecycle == AccountLifecycleAll {
		return AccountLifecycleNormal
	}
	return lifecycle
}

func ensureBlacklistedAccountNotRestored(account *Account, desiredStatus string, desiredSchedulable *bool) error {
	if account == nil || NormalizeAccountLifecycleInput(account.LifecycleState) != AccountLifecycleBlacklisted {
		return nil
	}

	if strings.EqualFold(strings.TrimSpace(desiredStatus), StatusActive) {
		return infraerrors.BadRequest("ACCOUNT_BLACKLISTED_RESTORE_FORBIDDEN", blacklistedRestoreForbiddenMessage)
	}
	if desiredSchedulable != nil && *desiredSchedulable {
		return infraerrors.BadRequest("ACCOUNT_BLACKLISTED_RESTORE_FORBIDDEN", blacklistedRestoreForbiddenMessage)
	}
	return nil
}

func (s *adminServiceImpl) BlacklistAccount(ctx context.Context, id int64, input *BlacklistAccountInput) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if account == nil {
		return nil, ErrAccountNotFound
	}
	if NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleBlacklisted {
		return account, nil
	}

	now := time.Now()
	purgeAt := now.Add(AccountBlacklistRetention)
	if err := s.accountRepo.MarkBlacklisted(ctx, id, manualBlacklistReasonCode, manualBlacklistReasonMessage, now, purgeAt); err != nil {
		return nil, err
	}
	updatedAccount, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if s.settingService != nil && input != nil && input.Feedback != nil {
		feedback := *input.Feedback
		if feedback.Platform == "" {
			feedback.Platform = RoutingPlatformForAccount(updatedAccount)
		}
		if feedback.Action == "" {
			feedback.Action = "blacklist"
		}
		if feedback.AdviceDecision == "" {
			feedback.AdviceDecision = string(BlacklistAdviceNotRecommended)
		}
		if err := s.settingService.RecordBlacklistRuleCandidate(ctx, feedback); err != nil {
			return nil, err
		}
	}
	return updatedAccount, nil
}

func (s *adminServiceImpl) RestoreBlacklistedAccount(ctx context.Context, id int64) (*Account, error) {
	account, err := s.accountRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if NormalizeAccountLifecycleInput(account.LifecycleState) != AccountLifecycleBlacklisted {
		return account, nil
	}
	if err := s.accountRepo.RestoreBlacklisted(ctx, id); err != nil {
		return nil, err
	}
	return s.accountRepo.GetByID(ctx, id)
}
