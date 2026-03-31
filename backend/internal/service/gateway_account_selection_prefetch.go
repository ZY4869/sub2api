package service

import (
	"context"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"log/slog"
	"time"
)

type usageLogWindowStatsBatchProvider interface {
	GetAccountWindowStatsBatch(ctx context.Context, accountIDs []int64, startTime time.Time) (map[int64]*usagestats.AccountStats, error)
}
type windowCostPrefetchContextKeyType struct{}

var windowCostPrefetchContextKey = windowCostPrefetchContextKeyType{}

func windowCostFromPrefetchContext(ctx context.Context, accountID int64) (float64, bool) {
	if ctx == nil || accountID <= 0 {
		return 0, false
	}
	m, ok := ctx.Value(windowCostPrefetchContextKey).(map[int64]float64)
	if !ok || len(m) == 0 {
		return 0, false
	}
	v, exists := m[accountID]
	return v, exists
}
func (s *GatewayService) withWindowCostPrefetch(ctx context.Context, accounts []Account) context.Context {
	if ctx == nil || len(accounts) == 0 || s.sessionLimitCache == nil || s.usageLogRepo == nil {
		return ctx
	}
	accountByID := make(map[int64]*Account)
	accountIDs := make([]int64, 0, len(accounts))
	for i := range accounts {
		account := &accounts[i]
		if !account.IsAnthropicOAuthOrSetupToken() {
			continue
		}
		if account.GetWindowCostLimit() <= 0 {
			continue
		}
		accountByID[account.ID] = account
		accountIDs = append(accountIDs, account.ID)
	}
	if len(accountIDs) == 0 {
		return ctx
	}
	costs := make(map[int64]float64, len(accountIDs))
	cacheValues, err := s.sessionLimitCache.GetWindowCostBatch(ctx, accountIDs)
	if err == nil {
		for accountID, cost := range cacheValues {
			costs[accountID] = cost
		}
		windowCostPrefetchCacheHitTotal.Add(int64(len(cacheValues)))
	} else {
		windowCostPrefetchErrorTotal.Add(1)
		logger.LegacyPrintf("service.gateway", "window_cost batch cache read failed: %v", err)
	}
	cacheMissCount := len(accountIDs) - len(costs)
	if cacheMissCount < 0 {
		cacheMissCount = 0
	}
	windowCostPrefetchCacheMissTotal.Add(int64(cacheMissCount))
	missingByStart := make(map[int64][]int64)
	startTimes := make(map[int64]time.Time)
	for _, accountID := range accountIDs {
		if _, ok := costs[accountID]; ok {
			continue
		}
		account := accountByID[accountID]
		if account == nil {
			continue
		}
		startTime := account.GetCurrentWindowStartTime()
		startKey := startTime.Unix()
		missingByStart[startKey] = append(missingByStart[startKey], accountID)
		startTimes[startKey] = startTime
	}
	if len(missingByStart) == 0 {
		return context.WithValue(ctx, windowCostPrefetchContextKey, costs)
	}
	batchReader, hasBatch := s.usageLogRepo.(usageLogWindowStatsBatchProvider)
	for startKey, ids := range missingByStart {
		startTime := startTimes[startKey]
		if hasBatch {
			windowCostPrefetchBatchSQLTotal.Add(1)
			queryStart := time.Now()
			statsByAccount, err := batchReader.GetAccountWindowStatsBatch(ctx, ids, startTime)
			if err == nil {
				slog.Debug("window_cost_batch_query_ok", "accounts", len(ids), "window_start", startTime.Format(time.RFC3339), "duration_ms", time.Since(queryStart).Milliseconds())
				for _, accountID := range ids {
					stats := statsByAccount[accountID]
					cost := 0.0
					if stats != nil {
						cost = stats.StandardCost
					}
					costs[accountID] = cost
					_ = s.sessionLimitCache.SetWindowCost(ctx, accountID, cost)
				}
				continue
			}
			windowCostPrefetchErrorTotal.Add(1)
			logger.LegacyPrintf("service.gateway", "window_cost batch db query failed: start=%s err=%v", startTime.Format(time.RFC3339), err)
		}
		windowCostPrefetchFallbackTotal.Add(int64(len(ids)))
		for _, accountID := range ids {
			stats, err := s.usageLogRepo.GetAccountWindowStats(ctx, accountID, startTime)
			if err != nil {
				windowCostPrefetchErrorTotal.Add(1)
				continue
			}
			cost := stats.StandardCost
			costs[accountID] = cost
			_ = s.sessionLimitCache.SetWindowCost(ctx, accountID, cost)
		}
	}
	return context.WithValue(ctx, windowCostPrefetchContextKey, costs)
}
func (s *GatewayService) isAccountSchedulableForQuota(account *Account) bool {
	if account == nil || !CanParticipateInAccountQuota(account) {
		return true
	}
	return !account.IsQuotaExceeded()
}
func (s *GatewayService) isAccountSchedulableForWindowCost(ctx context.Context, account *Account, isSticky bool) bool {
	if !account.IsAnthropicOAuthOrSetupToken() {
		return true
	}
	limit := account.GetWindowCostLimit()
	if limit <= 0 {
		return true
	}
	var currentCost float64
	if cost, ok := windowCostFromPrefetchContext(ctx, account.ID); ok {
		currentCost = cost
		goto checkSchedulability
	}
	if s.sessionLimitCache != nil {
		if cost, hit, err := s.sessionLimitCache.GetWindowCost(ctx, account.ID); err == nil && hit {
			currentCost = cost
			goto checkSchedulability
		}
	}
	{
		startTime := account.GetCurrentWindowStartTime()
		stats, err := s.usageLogRepo.GetAccountWindowStats(ctx, account.ID, startTime)
		if err != nil {
			return true
		}
		currentCost = stats.StandardCost
		if s.sessionLimitCache != nil {
			_ = s.sessionLimitCache.SetWindowCost(ctx, account.ID, currentCost)
		}
	}
checkSchedulability:
	schedulability := account.CheckWindowCostSchedulability(currentCost)
	switch schedulability {
	case WindowCostSchedulable:
		return true
	case WindowCostStickyOnly:
		return isSticky
	case WindowCostNotSchedulable:
		return false
	}
	return true
}

type rpmPrefetchContextKeyType struct{}

var rpmPrefetchContextKey = rpmPrefetchContextKeyType{}

func rpmFromPrefetchContext(ctx context.Context, accountID int64) (int, bool) {
	if v, ok := ctx.Value(rpmPrefetchContextKey).(map[int64]int); ok {
		count, found := v[accountID]
		return count, found
	}
	return 0, false
}
func (s *GatewayService) withRPMPrefetch(ctx context.Context, accounts []Account) context.Context {
	if s.rpmCache == nil {
		return ctx
	}
	var ids []int64
	for i := range accounts {
		if accounts[i].IsAnthropicOAuthOrSetupToken() && accounts[i].GetBaseRPM() > 0 {
			ids = append(ids, accounts[i].ID)
		}
	}
	if len(ids) == 0 {
		return ctx
	}
	counts, err := s.rpmCache.GetRPMBatch(ctx, ids)
	if err != nil {
		return ctx
	}
	return context.WithValue(ctx, rpmPrefetchContextKey, counts)
}
func (s *GatewayService) isAccountSchedulableForRPM(ctx context.Context, account *Account, isSticky bool) bool {
	if !account.IsAnthropicOAuthOrSetupToken() {
		return true
	}
	baseRPM := account.GetBaseRPM()
	if baseRPM <= 0 {
		return true
	}
	var currentRPM int
	if count, ok := rpmFromPrefetchContext(ctx, account.ID); ok {
		currentRPM = count
	} else if s.rpmCache != nil {
		if count, err := s.rpmCache.GetRPM(ctx, account.ID); err == nil {
			currentRPM = count
		}
	}
	schedulability := account.CheckRPMSchedulability(currentRPM)
	switch schedulability {
	case WindowCostSchedulable:
		return true
	case WindowCostStickyOnly:
		return isSticky
	case WindowCostNotSchedulable:
		return false
	}
	return true
}
func (s *GatewayService) IncrementAccountRPM(ctx context.Context, accountID int64) error {
	if s.rpmCache == nil {
		return nil
	}
	_, err := s.rpmCache.IncrementRPM(ctx, accountID)
	return err
}
func (s *GatewayService) getSchedulableAccount(ctx context.Context, accountID int64) (*Account, error) {
	if s.schedulerSnapshot != nil {
		return s.schedulerSnapshot.GetAccount(ctx, accountID)
	}
	return s.accountRepo.GetByID(ctx, accountID)
}
