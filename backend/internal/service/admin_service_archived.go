package service

import (
	"context"
	"fmt"
	"log/slog"
)

type archivedAccountGroupLister interface {
	ListArchivedGroups(ctx context.Context, filters ArchivedAccountGroupFilters) ([]ArchivedAccountGroupSummary, error)
}

type archivedAccountRestorer interface {
	RestoreArchived(ctx context.Context, id int64, restoreGroupIDs []int64, keepCurrentGroups bool) error
}

func (s *adminServiceImpl) ListArchivedGroups(ctx context.Context, filters ArchivedAccountGroupFilters) ([]ArchivedAccountGroupSummary, error) {
	lister, ok := s.accountRepo.(archivedAccountGroupLister)
	if !ok {
		return nil, fmt.Errorf("account repository does not support archived group summaries")
	}
	return lister.ListArchivedGroups(ctx, filters)
}

func (s *adminServiceImpl) UnarchiveAccounts(ctx context.Context, input *UnarchiveAccountsInput) (*UnarchiveAccountsResult, error) {
	result := &UnarchiveAccountsResult{}
	if input == nil || len(input.AccountIDs) == 0 {
		result.Results = []UnarchiveAccountResult{}
		return result, nil
	}

	restorer, ok := s.accountRepo.(archivedAccountRestorer)
	if !ok {
		return nil, fmt.Errorf("account repository does not support unarchive")
	}

	accountIDs := uniqueAccountIDs(input.AccountIDs)
	accounts, err := s.accountRepo.GetByIDs(ctx, accountIDs)
	if err != nil {
		return nil, err
	}

	accountByID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account != nil {
			accountByID[account.ID] = account
		}
	}

	slog.Info("admin_account_unarchive_started", "account_count", len(accountIDs))

	results := make([]UnarchiveAccountResult, 0, len(accountIDs))
	for _, accountID := range accountIDs {
		entry := UnarchiveAccountResult{AccountID: accountID}
		account := accountByID[accountID]
		if account == nil {
			entry.ErrorMessage = "account not found"
			result.FailedCount++
			results = append(results, entry)
			continue
		}
		if NormalizeAccountLifecycleInput(account.LifecycleState) != AccountLifecycleArchived {
			entry.ErrorMessage = "account is not archived"
			result.FailedCount++
			results = append(results, entry)
			continue
		}

		restoreGroupIDs, hasSnapshot := ParseArchiveRestoreGroupIDs(account.Extra)
		keepCurrentGroups := !hasSnapshot

		if err := restorer.RestoreArchived(ctx, accountID, restoreGroupIDs, keepCurrentGroups); err != nil {
			entry.ErrorMessage = err.Error()
			result.FailedCount++
			results = append(results, entry)
			continue
		}

		entry.Success = true
		entry.RestoredGroupIDs = append([]int64(nil), restoreGroupIDs...)
		entry.UsedFallbackCurrentGroup = keepCurrentGroups
		result.RestoredCount++
		if keepCurrentGroups {
			result.RestoredInPlaceCount++
		} else {
			result.RestoredToOriginalGroupCount++
		}
		results = append(results, entry)
	}

	result.Results = results

	slog.Info(
		"admin_account_unarchive_completed",
		"account_count", len(accountIDs),
		"restored_count", result.RestoredCount,
		"failed_count", result.FailedCount,
		"restored_to_original_group_count", result.RestoredToOriginalGroupCount,
		"restored_in_place_count", result.RestoredInPlaceCount,
	)

	return result, nil
}

func (s *adminServiceImpl) captureArchiveRestoreSnapshots(ctx context.Context, accounts map[int64]*Account, accountIDs []int64) error {
	for _, accountID := range accountIDs {
		account := accounts[accountID]
		if account == nil {
			continue
		}
		if NormalizeAccountLifecycleInput(account.LifecycleState) == AccountLifecycleArchived {
			continue
		}

		snapshot := BuildArchiveRestoreSnapshot(account)
		if len(snapshot) == 0 {
			continue
		}
		if err := s.accountRepo.UpdateExtra(ctx, accountID, snapshot); err != nil {
			return err
		}
	}
	return nil
}

func uniqueAccountIDs(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}
	seen := make(map[int64]struct{}, len(ids))
	out := make([]int64, 0, len(ids))
	for _, id := range ids {
		if id <= 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		out = append(out, id)
	}
	return out
}
