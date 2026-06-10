package service

import "time"

// isPeriodExpired 检查指定周期（自 periodStart 起经过 dur）是否已过期
func isPeriodExpired(periodStart time.Time, dur time.Duration) bool {
	if periodStart.IsZero() {
		return true // 从未使用过，视为过期（下次 increment 会初始化）
	}
	return time.Since(periodStart) >= dur
}

// IsQuotaExceeded 检查 API Key 账号配额是否已超限（任一维度超限即返回 true）
func (a *Account) IsQuotaExceeded() bool {
	// 总额度
	if limit := a.GetQuotaLimit(); limit > 0 && a.GetQuotaUsed() >= limit {
		return true
	}
	// 日额度（周期过期视为未超限，下次 increment 会重置）
	if limit := a.GetQuotaDailyLimit(); limit > 0 {
		start := a.getExtraTime("quota_daily_start")
		var expired bool
		if a.GetQuotaDailyResetMode() == "fixed" {
			expired = a.isFixedDailyPeriodExpired(start)
		} else {
			expired = isPeriodExpired(start, 24*time.Hour)
		}
		if !expired && a.GetQuotaDailyUsed() >= limit {
			return true
		}
	}
	// 周额度
	if limit := a.GetQuotaWeeklyLimit(); limit > 0 {
		start := a.getExtraTime("quota_weekly_start")
		var expired bool
		if a.GetQuotaWeeklyResetMode() == "fixed" {
			expired = a.isFixedWeeklyPeriodExpired(start)
		} else {
			expired = isPeriodExpired(start, 7*24*time.Hour)
		}
		if !expired && a.GetQuotaWeeklyUsed() >= limit {
			return true
		}
	}
	// 月额度
	if limit := a.GetQuotaMonthlyLimit(); limit > 0 {
		start := a.getExtraTime("quota_monthly_start")
		if !isPeriodExpired(start, 30*24*time.Hour) && a.GetQuotaMonthlyUsed() >= limit {
			return true
		}
	}
	if isQuotaCurrencyMapExceeded(a.GetQuotaLimitByCurrency(), a.GetQuotaUsedByCurrency()) {
		return true
	}
	if isQuotaCurrencyMapExceeded(a.GetQuotaDailyLimitByCurrency(), a.GetQuotaDailyUsedByCurrency()) {
		return true
	}
	if isQuotaCurrencyMapExceeded(a.GetQuotaWeeklyLimitByCurrency(), a.GetQuotaWeeklyUsedByCurrency()) {
		return true
	}
	if isQuotaCurrencyMapExceeded(a.GetQuotaMonthlyLimitByCurrency(), a.GetQuotaMonthlyUsedByCurrency()) {
		return true
	}
	return false
}

func isQuotaCurrencyMapExceeded(limits, used map[string]float64) bool {
	for currency, limit := range limits {
		if normalizeBillingCurrency(currency) == "" || limit <= 0 {
			continue
		}
		if used[normalizeBillingCurrency(currency)] >= limit {
			return true
		}
	}
	return false
}
