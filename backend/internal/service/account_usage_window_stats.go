package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	"github.com/Wei-Shaw/sub2api/internal/pkg/usagestats"
	"golang.org/x/sync/errgroup"
)

// addWindowStats 为 usage 数据添加窗口期统计
// 使用独立缓存（1 分钟），与 API 缓存分离
func (s *AccountUsageService) addWindowStats(ctx context.Context, account *Account, usage *UsageInfo, force bool) {
	// 修复：即使 FiveHour 为 nil，也要尝试获取统计数据
	// 因为 SevenDay/SevenDaySonnet 可能需要
	if usage.FiveHour == nil && usage.SevenDay == nil && usage.SevenDaySonnet == nil {
		return
	}

	// 检查窗口统计缓存（1 分钟）
	var windowStats *WindowStats
	if !force {
		if cached, ok := s.cache.windowStatsCache.Load(account.ID); ok {
			if cache, ok := cached.(*windowStatsCache); ok && time.Since(cache.timestamp) < windowStatsCacheTTL {
				windowStats = cache.stats
			}
		}
	}

	// 如果没有缓存，从数据库查询
	if windowStats == nil {
		// 使用统一的窗口开始时间计算逻辑（考虑窗口过期情况）
		startTime := account.GetCurrentWindowStartTime()

		stats, err := s.usageLogRepo.GetAccountWindowStats(ctx, account.ID, startTime)
		if err != nil {
			log.Printf("Failed to get window stats for account %d: %v", account.ID, err)
			return
		}

		windowStats = &WindowStats{
			Requests:     stats.Requests,
			Tokens:       stats.Tokens,
			Cost:         stats.Cost,
			StandardCost: stats.StandardCost,
			UserCost:     stats.UserCost,
		}

		// 缓存窗口统计（1 分钟）
		s.cache.windowStatsCache.Store(account.ID, &windowStatsCache{
			stats:     windowStats,
			timestamp: time.Now(),
		})
	}

	// 为 FiveHour 添加 WindowStats（5h 窗口统计）
	if usage.FiveHour != nil {
		usage.FiveHour.WindowStats = windowStats
	}
}

// GetTodayStats 获取账号今日统计
func (s *AccountUsageService) GetTodayStats(ctx context.Context, accountID int64) (*WindowStats, error) {
	statsByAccount, err := s.GetTodayStatsBatch(ctx, []int64{accountID})
	if err != nil {
		return nil, fmt.Errorf("get today stats failed: %w", err)
	}
	if stats := statsByAccount[accountID]; stats != nil {
		return stats, nil
	}
	return windowStatsFromTodayBreakdown(nil), nil
}

// GetTodayStatsBatch 批量获取账号今日统计，优先走批量 SQL，失败时回退单账号查询。
func (s *AccountUsageService) GetTodayStatsBatch(ctx context.Context, accountIDs []int64) (map[int64]*WindowStats, error) {
	uniqueIDs := make([]int64, 0, len(accountIDs))
	seen := make(map[int64]struct{}, len(accountIDs))
	for _, accountID := range accountIDs {
		if accountID <= 0 {
			continue
		}
		if _, exists := seen[accountID]; exists {
			continue
		}
		seen[accountID] = struct{}{}
		uniqueIDs = append(uniqueIDs, accountID)
	}

	result := make(map[int64]*WindowStats, len(uniqueIDs))
	if len(uniqueIDs) == 0 {
		return result, nil
	}

	startTime := timezone.Today()
	weekStart := startTime.AddDate(0, 0, -6)
	if breakdownReader, ok := s.usageLogRepo.(accountTodayStatsBreakdownBatchReader); ok {
		breakdownByAccount, err := breakdownReader.GetAccountTodayStatsBreakdownBatch(ctx, uniqueIDs, startTime, weekStart)
		if err == nil {
			for _, accountID := range uniqueIDs {
				result[accountID] = windowStatsFromTodayBreakdown(breakdownByAccount[accountID])
			}
			return result, nil
		}
	}

	if batchReader, ok := s.usageLogRepo.(accountWindowStatsBatchReader); ok {
		statsByAccount, err := batchReader.GetAccountWindowStatsBatch(ctx, uniqueIDs, startTime)
		if err == nil {
			for _, accountID := range uniqueIDs {
				result[accountID] = windowStatsFromAccountStats(statsByAccount[accountID])
			}
			return result, nil
		}
	}

	var mu sync.Mutex
	g, gctx := errgroup.WithContext(ctx)
	g.SetLimit(8)

	for _, accountID := range uniqueIDs {
		id := accountID
		g.Go(func() error {
			stats, err := s.usageLogRepo.GetAccountWindowStats(gctx, id, startTime)
			if err != nil {
				return nil
			}
			mu.Lock()
			result[id] = windowStatsFromAccountStats(stats)
			mu.Unlock()
			return nil
		})
	}

	_ = g.Wait()

	for _, accountID := range uniqueIDs {
		if _, ok := result[accountID]; !ok {
			result[accountID] = &WindowStats{}
		}
	}
	return result, nil
}

func windowStatsFromTodayBreakdown(stats *usagestats.AccountTodayStatsBreakdown) *WindowStats {
	if stats == nil {
		return &WindowStats{
			SuccessRate: 100,
			Weekly:      &WindowStats{SuccessRate: 100},
			Total:       &WindowStats{SuccessRate: 100},
		}
	}
	today := windowStatsFromAccountStats(&stats.Today)
	today.Weekly = windowStatsFromAccountStats(&stats.Weekly)
	today.Total = windowStatsFromAccountStats(&stats.Total)
	return today
}

func windowStatsFromAccountStats(stats *usagestats.AccountStats) *WindowStats {
	if stats == nil {
		return &WindowStats{SuccessRate: 100}
	}
	successRate := stats.SuccessRate
	if stats.Requests == 0 && successRate == 0 {
		successRate = 100
	}
	return &WindowStats{
		Requests:          stats.Requests,
		Tokens:            stats.Tokens,
		Cost:              stats.Cost,
		StandardCost:      stats.StandardCost,
		UserCost:          stats.UserCost,
		SuccessRate:       successRate,
		AverageDurationMs: stats.AverageDurationMs,
	}
}

func (s *AccountUsageService) GetAccountUsageStats(ctx context.Context, accountID int64, startTime, endTime time.Time) (*usagestats.AccountUsageStatsResponse, error) {
	stats, err := s.usageLogRepo.GetAccountUsageStats(ctx, accountID, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("get account usage stats failed: %w", err)
	}
	return stats, nil
}

// GetAccountWindowStats 获取账号在指定时间窗口内的使用统计
// 用于账号列表页面显示当前窗口费用
func (s *AccountUsageService) GetAccountWindowStats(ctx context.Context, accountID int64, startTime time.Time) (*usagestats.AccountStats, error) {
	return s.usageLogRepo.GetAccountWindowStats(ctx, accountID, startTime)
}
