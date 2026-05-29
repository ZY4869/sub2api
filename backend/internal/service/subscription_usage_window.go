package service

import (
	"context"
	"time"
)

// startOfDay 返回给定时间所在日期的零点（保持原时区）
func startOfDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

// CheckAndActivateWindow 检查并激活使用窗口（首次使用时）
func (s *SubscriptionService) CheckAndActivateWindow(ctx context.Context, sub *UserSubscription) error {
	if sub.IsWindowActivated() {
		return nil
	}

	// 使用当天零点作为窗口起始时间
	windowStart := startOfDay(time.Now())
	return s.userSubRepo.ActivateWindows(ctx, sub.ID, windowStart)
}

// AdminResetQuota manually resets the daily, weekly, and/or monthly usage windows.
// Uses startOfDay(now) as the new window start, matching automatic resets.
func (s *SubscriptionService) AdminResetQuota(ctx context.Context, subscriptionID int64, resetDaily, resetWeekly, resetMonthly bool) (*UserSubscription, error) {
	if !resetDaily && !resetWeekly && !resetMonthly {
		return nil, ErrInvalidInput
	}
	sub, err := s.userSubRepo.GetByID(ctx, subscriptionID)
	if err != nil {
		return nil, err
	}
	windowStart := startOfDay(time.Now())
	if resetDaily {
		if err := s.userSubRepo.ResetDailyUsage(ctx, sub.ID, windowStart); err != nil {
			return nil, err
		}
	}
	if resetWeekly {
		if err := s.userSubRepo.ResetWeeklyUsage(ctx, sub.ID, windowStart); err != nil {
			return nil, err
		}
	}
	if resetMonthly {
		if err := s.userSubRepo.ResetMonthlyUsage(ctx, sub.ID, windowStart); err != nil {
			return nil, err
		}
	}
	// Invalidate L1 ristretto cache. Ristretto's Del() is asynchronous by design,
	// so call Wait() immediately after to flush pending operations and guarantee
	// the deleted key is not returned on the very next Get() call.
	s.InvalidateSubCache(sub.UserID, sub.GroupID)
	if s.subCacheL1 != nil {
		s.subCacheL1.Wait()
	}
	if s.billingCacheService != nil {
		_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
	}
	// Return the refreshed subscription from DB
	return s.userSubRepo.GetByID(ctx, subscriptionID)
}

// CheckAndResetWindows 检查并重置过期的窗口
func (s *SubscriptionService) CheckAndResetWindows(ctx context.Context, sub *UserSubscription) error {
	// 使用当天零点作为新窗口起始时间
	windowStart := startOfDay(time.Now())
	needsInvalidateCache := false

	// 日窗口重置（24小时）
	if sub.NeedsDailyReset() {
		if err := s.userSubRepo.ResetDailyUsage(ctx, sub.ID, windowStart); err != nil {
			return err
		}
		sub.DailyWindowStart = &windowStart
		sub.DailyUsageUSD = 0
		sub.DailyUsageByCurrency = nil
		needsInvalidateCache = true
	}

	// 周窗口重置（7天）
	if sub.NeedsWeeklyReset() {
		if err := s.userSubRepo.ResetWeeklyUsage(ctx, sub.ID, windowStart); err != nil {
			return err
		}
		sub.WeeklyWindowStart = &windowStart
		sub.WeeklyUsageUSD = 0
		sub.WeeklyUsageByCurrency = nil
		needsInvalidateCache = true
	}

	// 月窗口重置（30天）
	if sub.NeedsMonthlyReset() {
		if err := s.userSubRepo.ResetMonthlyUsage(ctx, sub.ID, windowStart); err != nil {
			return err
		}
		sub.MonthlyWindowStart = &windowStart
		sub.MonthlyUsageUSD = 0
		sub.MonthlyUsageByCurrency = nil
		needsInvalidateCache = true
	}

	// 如果有窗口被重置，失效缓存以保持一致性
	if needsInvalidateCache {
		s.InvalidateSubCache(sub.UserID, sub.GroupID)
		if s.billingCacheService != nil {
			_ = s.billingCacheService.InvalidateSubscription(ctx, sub.UserID, sub.GroupID)
		}
	}

	return nil
}

// CheckUsageLimits 检查使用限额（返回错误如果超限）
// 用于中间件的快速预检查，additionalCost 通常为 0
func (s *SubscriptionService) CheckUsageLimits(ctx context.Context, sub *UserSubscription, group *Group, additionalCost float64) error {
	if !sub.CheckDailyLimit(group, additionalCost) {
		return ErrDailyLimitExceeded
	}
	if !sub.CheckWeeklyLimit(group, additionalCost) {
		return ErrWeeklyLimitExceeded
	}
	if !sub.CheckMonthlyLimit(group, additionalCost) {
		return ErrMonthlyLimitExceeded
	}
	return nil
}

// ValidateAndCheckLimits 合并验证+限额检查（中间件热路径专用）
// 仅做内存检查，不触发 DB 写入。窗口重置的 DB 写入由 DoWindowMaintenance 异步完成。
// 返回 needsMaintenance 表示是否需要异步执行窗口维护。
func (s *SubscriptionService) ValidateAndCheckLimits(sub *UserSubscription, group *Group) (needsMaintenance bool, err error) {
	// 1. 验证订阅状态
	if sub.Status == SubscriptionStatusExpired {
		return false, ErrSubscriptionExpired
	}
	if sub.Status == SubscriptionStatusSuspended {
		return false, ErrSubscriptionSuspended
	}
	if sub.IsExpired() {
		return false, ErrSubscriptionExpired
	}

	// 2. 内存中修正过期窗口的用量，确保 CheckUsageLimits 不会误拒绝用户
	//    实际的 DB 窗口重置由 DoWindowMaintenance 异步完成
	if sub.NeedsDailyReset() {
		sub.DailyUsageUSD = 0
		sub.DailyUsageByCurrency = nil
		needsMaintenance = true
	}
	if sub.NeedsWeeklyReset() {
		sub.WeeklyUsageUSD = 0
		sub.WeeklyUsageByCurrency = nil
		needsMaintenance = true
	}
	if sub.NeedsMonthlyReset() {
		sub.MonthlyUsageUSD = 0
		sub.MonthlyUsageByCurrency = nil
		needsMaintenance = true
	}
	if !sub.IsWindowActivated() {
		needsMaintenance = true
	}

	// 3. 检查用量限额
	if !sub.CheckDailyLimit(group, 0) {
		return needsMaintenance, ErrDailyLimitExceeded
	}
	if !sub.CheckWeeklyLimit(group, 0) {
		return needsMaintenance, ErrWeeklyLimitExceeded
	}
	if !sub.CheckMonthlyLimit(group, 0) {
		return needsMaintenance, ErrMonthlyLimitExceeded
	}

	return needsMaintenance, nil
}

// RecordUsage 记录使用量到订阅
func (s *SubscriptionService) RecordUsage(ctx context.Context, subscriptionID int64, costUSD float64) error {
	return s.userSubRepo.IncrementUsage(ctx, subscriptionID, costUSD)
}
