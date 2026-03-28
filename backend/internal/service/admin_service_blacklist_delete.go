package service

import "context"

const (
	blacklistedBatchDeleteNotFoundReason       = "account not found"
	blacklistedBatchDeleteNotBlacklistedReason = "account is not blacklisted"
)

func (s *adminServiceImpl) BatchDeleteBlacklistedAccounts(ctx context.Context, ids []int64, deleteAll bool) (*BlacklistedBatchDeleteResult, error) {
	targetIDs, initialFailures, err := s.resolveBlacklistedDeleteTargets(ctx, ids, deleteAll)
	if err != nil {
		return nil, err
	}

	result := &BlacklistedBatchDeleteResult{
		DeletedIDs: make([]int64, 0, len(targetIDs)),
		Failed:     make([]BlacklistedBatchDeleteFailure, 0, len(initialFailures)+len(targetIDs)),
	}
	result.Failed = append(result.Failed, initialFailures...)

	for _, id := range targetIDs {
		if err := s.DeleteAccount(ctx, id); err != nil {
			result.Failed = append(result.Failed, BlacklistedBatchDeleteFailure{
				ID:     id,
				Reason: err.Error(),
			})
			continue
		}
		result.DeletedIDs = append(result.DeletedIDs, id)
	}

	result.DeletedCount = len(result.DeletedIDs)
	result.FailedCount = len(result.Failed)
	return result, nil
}

func (s *adminServiceImpl) resolveBlacklistedDeleteTargets(ctx context.Context, ids []int64, deleteAll bool) ([]int64, []BlacklistedBatchDeleteFailure, error) {
	if deleteAll {
		targetIDs, err := s.accountRepo.ListBlacklistedIDs(ctx)
		if err != nil {
			return nil, nil, err
		}
		return normalizePositiveUniqueInt64s(targetIDs), nil, nil
	}

	normalizedIDs := normalizePositiveUniqueInt64s(ids)
	if len(normalizedIDs) == 0 {
		return nil, nil, nil
	}

	accounts, err := s.accountRepo.GetByIDs(ctx, normalizedIDs)
	if err != nil {
		return nil, nil, err
	}

	accountByID := make(map[int64]*Account, len(accounts))
	for _, account := range accounts {
		if account == nil {
			continue
		}
		accountByID[account.ID] = account
	}

	targetIDs := make([]int64, 0, len(normalizedIDs))
	failures := make([]BlacklistedBatchDeleteFailure, 0)
	for _, id := range normalizedIDs {
		account := accountByID[id]
		switch {
		case account == nil:
			failures = append(failures, BlacklistedBatchDeleteFailure{ID: id, Reason: blacklistedBatchDeleteNotFoundReason})
		case NormalizeAccountLifecycleInput(account.LifecycleState) != AccountLifecycleBlacklisted:
			failures = append(failures, BlacklistedBatchDeleteFailure{ID: id, Reason: blacklistedBatchDeleteNotBlacklistedReason})
		default:
			targetIDs = append(targetIDs, id)
		}
	}

	return targetIDs, failures, nil
}

func normalizePositiveUniqueInt64s(ids []int64) []int64 {
	if len(ids) == 0 {
		return nil
	}

	out := make([]int64, 0, len(ids))
	seen := make(map[int64]struct{}, len(ids))
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
