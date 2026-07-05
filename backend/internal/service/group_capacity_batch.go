package service

import (
	"context"
	"time"
)

// GroupAccountCapacityRow is the lightweight account projection needed for
// capacity summary aggregation.
type GroupAccountCapacityRow struct {
	GroupID             int64
	AccountID           int64
	Concurrency         int
	Extra               map[string]any
	SessionWindowStart  *time.Time
	SessionWindowEnd    *time.Time
	SessionWindowStatus string
}

type groupCapacityActiveGroupIDLister interface {
	ListActiveIDs(ctx context.Context) ([]int64, error)
}

type groupCapacityAccountLister interface {
	ListSchedulableCapacityByGroupIDs(ctx context.Context, groupIDs []int64) ([]GroupAccountCapacityRow, error)
}

func (s *GroupCapacityService) getGroupCapacitiesBatch(ctx context.Context, groupIDs []int64, lister groupCapacityAccountLister) ([]GroupCapacitySummary, error) {
	results := initGroupCapacityResults(groupIDs)
	if len(groupIDs) == 0 {
		return results, nil
	}

	rows, err := lister.ListSchedulableCapacityByGroupIDs(ctx, groupIDs)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return results, nil
	}

	groupIndex := indexGroupCapacityResults(results)
	refs, accountIDs, sessionTimeouts := collectGroupCapacityRows(rows, groupIndex, results)
	if len(accountIDs) == 0 {
		return results, nil
	}

	concurrencyMap := s.getGroupCapacityConcurrencyMap(ctx, accountIDs)
	sessionsMap := s.getGroupCapacitySessionsMap(ctx, refs, groupIndex, results, sessionTimeouts)
	rpmMap := s.getGroupCapacityRPMMap(ctx, refs, groupIndex, results)
	applyGroupCapacityRuntimeCounts(results, groupIndex, refs, concurrencyMap, sessionsMap, rpmMap)
	return results, nil
}

func initGroupCapacityResults(groupIDs []int64) []GroupCapacitySummary {
	results := make([]GroupCapacitySummary, len(groupIDs))
	for i, groupID := range groupIDs {
		results[i].GroupID = groupID
	}
	return results
}

func indexGroupCapacityResults(results []GroupCapacitySummary) map[int64]int {
	index := make(map[int64]int, len(results))
	for i := range results {
		index[results[i].GroupID] = i
	}
	return index
}

func collectGroupCapacityRows(rows []GroupAccountCapacityRow, groupIndex map[int64]int, results []GroupCapacitySummary) ([]groupCapacityAccountRef, []int64, map[int64]time.Duration) {
	refs := make([]groupCapacityAccountRef, 0, len(rows))
	seenGroupAccount := make(map[groupCapacityAccountRef]struct{}, len(rows))
	accountIDSet := make(map[int64]struct{}, len(rows))
	accountIDs := make([]int64, 0, len(rows))
	sessionTimeouts := make(map[int64]time.Duration)

	for _, row := range rows {
		idx, ok := groupIndex[row.GroupID]
		if !ok || row.AccountID <= 0 {
			continue
		}

		ref := groupCapacityAccountRef{groupID: row.GroupID, accountID: row.AccountID}
		if _, ok := seenGroupAccount[ref]; ok {
			continue
		}
		seenGroupAccount[ref] = struct{}{}
		refs = append(refs, ref)

		if _, ok := accountIDSet[row.AccountID]; !ok {
			accountIDSet[row.AccountID] = struct{}{}
			accountIDs = append(accountIDs, row.AccountID)
		}

		acc := groupCapacityAccountFromRow(row)
		results[idx].ConcurrencyMax += acc.Concurrency
		if maxSessions := acc.GetMaxSessions(); maxSessions > 0 {
			results[idx].SessionsMax += maxSessions
			sessionTimeouts[acc.ID] = groupCapacitySessionTimeout(acc)
		}
		if rpm := acc.GetBaseRPM(); rpm > 0 {
			results[idx].RPMMax += rpm
		}
	}
	return refs, accountIDs, sessionTimeouts
}

func groupCapacityAccountFromRow(row GroupAccountCapacityRow) Account {
	return Account{
		ID:                  row.AccountID,
		Concurrency:         row.Concurrency,
		Extra:               row.Extra,
		SessionWindowStart:  row.SessionWindowStart,
		SessionWindowEnd:    row.SessionWindowEnd,
		SessionWindowStatus: row.SessionWindowStatus,
	}
}

func groupCapacitySessionTimeout(account Account) time.Duration {
	timeout := time.Duration(account.GetSessionIdleTimeoutMinutes()) * time.Minute
	if timeout <= 0 {
		return 5 * time.Minute
	}
	return timeout
}

func (s *GroupCapacityService) getGroupCapacityConcurrencyMap(ctx context.Context, accountIDs []int64) map[int64]int {
	if s == nil || s.concurrencyService == nil {
		return map[int64]int{}
	}
	out, _ := s.concurrencyService.GetAccountConcurrencyBatch(ctx, accountIDs)
	if out == nil {
		return map[int64]int{}
	}
	return out
}

func (s *GroupCapacityService) getGroupCapacitySessionsMap(ctx context.Context, refs []groupCapacityAccountRef, groupIndex map[int64]int, results []GroupCapacitySummary, sessionTimeouts map[int64]time.Duration) map[int64]int {
	accountIDs := accountIDsForGroupsWithLimit(refs, groupIndex, results, func(summary GroupCapacitySummary) bool {
		return summary.SessionsMax > 0
	})
	if len(accountIDs) == 0 || s == nil || s.sessionLimitCache == nil {
		return nil
	}
	out, _ := s.sessionLimitCache.GetActiveSessionCountBatch(ctx, accountIDs, sessionTimeouts)
	return out
}

func (s *GroupCapacityService) getGroupCapacityRPMMap(ctx context.Context, refs []groupCapacityAccountRef, groupIndex map[int64]int, results []GroupCapacitySummary) map[int64]int {
	accountIDs := accountIDsForGroupsWithLimit(refs, groupIndex, results, func(summary GroupCapacitySummary) bool {
		return summary.RPMMax > 0
	})
	if len(accountIDs) == 0 || s == nil || s.rpmCache == nil {
		return nil
	}
	out, _ := s.rpmCache.GetRPMBatch(ctx, accountIDs)
	return out
}

func applyGroupCapacityRuntimeCounts(results []GroupCapacitySummary, groupIndex map[int64]int, refs []groupCapacityAccountRef, concurrencyMap, sessionsMap, rpmMap map[int64]int) {
	for _, ref := range refs {
		idx := groupIndex[ref.groupID]
		results[idx].ConcurrencyUsed += concurrencyMap[ref.accountID]
		if sessionsMap != nil && results[idx].SessionsMax > 0 {
			results[idx].SessionsUsed += sessionsMap[ref.accountID]
		}
		if rpmMap != nil && results[idx].RPMMax > 0 {
			results[idx].RPMUsed += rpmMap[ref.accountID]
		}
	}
}

func accountIDsForGroupsWithLimit(refs []groupCapacityAccountRef, groupIndex map[int64]int, summaries []GroupCapacitySummary, include func(GroupCapacitySummary) bool) []int64 {
	seen := make(map[int64]struct{})
	accountIDs := make([]int64, 0)
	for _, ref := range refs {
		idx, ok := groupIndex[ref.groupID]
		if !ok || !include(summaries[idx]) {
			continue
		}
		if _, ok := seen[ref.accountID]; ok {
			continue
		}
		seen[ref.accountID] = struct{}{}
		accountIDs = append(accountIDs, ref.accountID)
	}
	return accountIDs
}
