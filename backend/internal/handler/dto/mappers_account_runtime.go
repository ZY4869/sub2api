package dto

import "github.com/Wei-Shaw/sub2api/internal/service"

func applyAccountAnthropicRuntime(a *service.Account, out *Account) {
	if !a.IsAnthropicOAuthOrSetupToken() {
		return
	}
	if limit := a.GetWindowCostLimit(); limit > 0 {
		out.WindowCostLimit = &limit
	}
	if reserve := a.GetWindowCostStickyReserve(); reserve > 0 {
		out.WindowCostStickyReserve = &reserve
	}
	if maxSessions := a.GetMaxSessions(); maxSessions > 0 {
		out.MaxSessions = &maxSessions
	}
	if idleTimeout := a.GetSessionIdleTimeoutMinutes(); idleTimeout > 0 {
		out.SessionIdleTimeoutMin = &idleTimeout
	}
	if rpm := a.GetBaseRPM(); rpm > 0 {
		out.BaseRPM = &rpm
		strategy := a.GetRPMStrategy()
		out.RPMStrategy = &strategy
		buffer := a.GetRPMStickyBuffer()
		out.RPMStickyBuffer = &buffer
	}
	if mode := a.GetUserMsgQueueMode(); mode != "" {
		out.UserMsgQueueMode = &mode
	}
	if a.IsTLSFingerprintEnabled() {
		enabled := true
		out.EnableTLSFingerprint = &enabled
	}
	if profileID := a.GetTLSFingerprintProfileID(); profileID != 0 {
		out.TLSFingerprintProfileID = &profileID
	}
	if a.IsSessionIDMaskingEnabled() {
		enabled := true
		out.EnableSessionIDMasking = &enabled
	}
	if a.IsCacheTTLOverrideEnabled() {
		enabled := true
		out.CacheTTLOverrideEnabled = &enabled
		target := a.GetCacheTTLOverrideTarget()
		out.CacheTTLOverrideTarget = &target
	}
	if a.IsCustomBaseURLEnabled() {
		enabled := true
		out.CustomBaseURLEnabled = &enabled
		if customURL := a.GetCustomBaseURL(); customURL != "" {
			out.CustomBaseURL = &customURL
		}
	}
}

func applyAccountProtocolGatewayMimic(a *service.Account, out *Account) {
	if !service.SupportsProtocolGatewayClaudeClientMimic(a) {
		return
	}
	mimicEnabled := service.IsClaudeClientMimicEnabled(a, service.PlatformAnthropic)
	out.ClaudeCodeMimicEnabled = &mimicEnabled
	tlsEnabled := a.IsTLSFingerprintEnabled()
	out.EnableTLSFingerprint = &tlsEnabled
	if profileID := a.GetTLSFingerprintProfileID(); profileID != 0 {
		out.TLSFingerprintProfileID = &profileID
	}
	sessionMaskingEnabled := a.IsSessionIDMaskingEnabled()
	out.EnableSessionIDMasking = &sessionMaskingEnabled
}

func applyAccountQuota(a *service.Account, out *Account) {
	if !service.ShouldExposeAccountQuota(a) {
		return
	}
	if limit := a.GetQuotaLimit(); limit > 0 {
		out.QuotaLimit = &limit
		used := a.GetQuotaUsed()
		out.QuotaUsed = &used
	}
	out.QuotaLimitByCurrency = cloneUsageCostByCurrency(a.GetQuotaLimitByCurrency())
	out.QuotaUsedByCurrency = cloneUsageCostByCurrency(a.GetQuotaUsedByCurrency())
	if limit := a.GetQuotaDailyLimit(); limit > 0 {
		out.QuotaDailyLimit = &limit
		used := a.GetQuotaDailyUsed()
		out.QuotaDailyUsed = &used
	}
	out.QuotaDailyLimitByCurrency = cloneUsageCostByCurrency(a.GetQuotaDailyLimitByCurrency())
	out.QuotaDailyUsedByCurrency = cloneUsageCostByCurrency(a.GetQuotaDailyUsedByCurrency())
	if limit := a.GetQuotaWeeklyLimit(); limit > 0 {
		out.QuotaWeeklyLimit = &limit
		used := a.GetQuotaWeeklyUsed()
		out.QuotaWeeklyUsed = &used
	}
	out.QuotaWeeklyLimitByCurrency = cloneUsageCostByCurrency(a.GetQuotaWeeklyLimitByCurrency())
	out.QuotaWeeklyUsedByCurrency = cloneUsageCostByCurrency(a.GetQuotaWeeklyUsedByCurrency())
	if limit := a.GetQuotaMonthlyLimit(); limit > 0 {
		out.QuotaMonthlyLimit = &limit
		used := a.GetQuotaMonthlyUsed()
		out.QuotaMonthlyUsed = &used
	}
	out.QuotaMonthlyLimitByCurrency = cloneUsageCostByCurrency(a.GetQuotaMonthlyLimitByCurrency())
	out.QuotaMonthlyUsedByCurrency = cloneUsageCostByCurrency(a.GetQuotaMonthlyUsedByCurrency())
	if mode := a.GetQuotaDailyResetMode(); mode == "fixed" {
		out.QuotaDailyResetMode = &mode
		hour := a.GetQuotaDailyResetHour()
		out.QuotaDailyResetHour = &hour
	}
	if mode := a.GetQuotaWeeklyResetMode(); mode == "fixed" {
		out.QuotaWeeklyResetMode = &mode
		day := a.GetQuotaWeeklyResetDay()
		out.QuotaWeeklyResetDay = &day
		hour := a.GetQuotaWeeklyResetHour()
		out.QuotaWeeklyResetHour = &hour
	}
	if a.GetQuotaDailyResetMode() == "fixed" || a.GetQuotaWeeklyResetMode() == "fixed" {
		tz := a.GetQuotaResetTimezone()
		out.QuotaResetTimezone = &tz
	}
	if a.Extra != nil {
		if v, ok := a.Extra["quota_daily_reset_at"].(string); ok && v != "" {
			out.QuotaDailyResetAt = &v
		}
		if v, ok := a.Extra["quota_weekly_reset_at"].(string); ok && v != "" {
			out.QuotaWeeklyResetAt = &v
		}
		if v, ok := a.Extra["quota_monthly_reset_at"].(string); ok && v != "" {
			out.QuotaMonthlyResetAt = &v
		}
	}
}

func applyAccountGeminiBatch(a *service.Account, out *Account) {
	if !a.IsGemini() {
		return
	}
	batchArchiveEnabled := a.IsBatchArchiveEnabled()
	out.BatchArchiveEnabled = &batchArchiveEnabled
	autoPrefetchEnabled := a.IsBatchArchiveAutoPrefetchEnabled()
	out.BatchArchiveAutoPrefetchEnabled = &autoPrefetchEnabled
	retentionDays := a.GetBatchArchiveRetentionDays()
	out.BatchArchiveRetentionDays = &retentionDays
	billingMode := a.GetBatchArchiveBillingMode()
	out.BatchArchiveBillingMode = &billingMode
	downloadPrice := a.GetBatchArchiveDownloadPriceUSD()
	out.BatchArchiveDownloadPriceUSD = &downloadPrice
	allowOverflow := a.AllowVertexBatchOverflow()
	out.AllowVertexBatchOverflow = &allowOverflow
	acceptOverflow := a.AcceptAIStudioBatchOverflow()
	out.AcceptAIStudioBatchOverflow = &acceptOverflow
}
