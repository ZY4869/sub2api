package handler

import (
	"context"
	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
	"net/http"
	"time"
)

func (h *GatewayHandler) Usage(c *gin.Context) {
	apiKey, ok := middleware2.GetAPIKeyFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		h.errorResponse(c, http.StatusUnauthorized, "authentication_error", "Invalid API key")
		return
	}
	ctx := c.Request.Context()
	startTime, endTime := h.parseUsageDateRange(c)
	usageData := h.buildUsageData(ctx, apiKey.ID)
	var modelStats any
	if h.usageService != nil {
		if stats, err := h.usageService.GetAPIKeyModelStats(ctx, apiKey.ID, startTime, endTime); err == nil && len(stats) > 0 {
			modelStats = stats
		}
	}
	isQuotaLimited := apiKey.Quota > 0 || apiKey.HasRateLimits()
	if isQuotaLimited {
		h.usageQuotaLimited(c, ctx, apiKey, usageData, modelStats)
		return
	}
	h.usageUnrestricted(c, ctx, apiKey, subject, usageData, modelStats)
}
func (h *GatewayHandler) parseUsageDateRange(c *gin.Context) (time.Time, time.Time) {
	now := timezone.Now()
	endTime := now
	startTime := now.AddDate(0, 0, -30)
	if s := c.Query("start_date"); s != "" {
		if t, err := timezone.ParseInLocation("2006-01-02", s); err == nil {
			startTime = t
		}
	}
	if s := c.Query("end_date"); s != "" {
		if t, err := timezone.ParseInLocation("2006-01-02", s); err == nil {
			endTime = t.Add(24*time.Hour - time.Second)
		}
	}
	return startTime, endTime
}
func (h *GatewayHandler) buildUsageData(ctx context.Context, apiKeyID int64) gin.H {
	if h.usageService == nil {
		return nil
	}
	dashStats, err := h.usageService.GetAPIKeyDashboardStats(ctx, apiKeyID)
	if err != nil || dashStats == nil {
		return nil
	}
	return gin.H{"today": gin.H{"requests": dashStats.TodayRequests, "input_tokens": dashStats.TodayInputTokens, "output_tokens": dashStats.TodayOutputTokens, "cache_creation_tokens": dashStats.TodayCacheCreationTokens, "cache_read_tokens": dashStats.TodayCacheReadTokens, "total_tokens": dashStats.TodayTokens, "cost": dashStats.TodayCost, "actual_cost": dashStats.TodayActualCost}, "total": gin.H{"requests": dashStats.TotalRequests, "input_tokens": dashStats.TotalInputTokens, "output_tokens": dashStats.TotalOutputTokens, "cache_creation_tokens": dashStats.TotalCacheCreationTokens, "cache_read_tokens": dashStats.TotalCacheReadTokens, "total_tokens": dashStats.TotalTokens, "cost": dashStats.TotalCost, "actual_cost": dashStats.TotalActualCost}, "average_duration_ms": dashStats.AverageDurationMs, "rpm": dashStats.Rpm, "tpm": dashStats.Tpm}
}
func (h *GatewayHandler) usageQuotaLimited(c *gin.Context, ctx context.Context, apiKey *service.APIKey, usageData gin.H, modelStats any) {
	resp := gin.H{"mode": "quota_limited", "isValid": apiKey.Status == service.StatusAPIKeyActive || apiKey.Status == service.StatusAPIKeyQuotaExhausted || apiKey.Status == service.StatusAPIKeyExpired, "status": apiKey.Status}
	if apiKey.Quota > 0 {
		remaining := apiKey.GetQuotaRemaining()
		resp["quota"] = gin.H{"limit": apiKey.Quota, "used": apiKey.QuotaUsed, "remaining": remaining, "unit": "USD"}
		resp["remaining"] = remaining
		resp["unit"] = "USD"
	}
	if apiKey.HasRateLimits() && h.apiKeyService != nil {
		rateLimitData, err := h.apiKeyService.GetRateLimitData(ctx, apiKey.ID)
		if err == nil && rateLimitData != nil {
			var rateLimits []gin.H
			if apiKey.RateLimit5h > 0 {
				used := rateLimitData.EffectiveUsage5h()
				entry := gin.H{"window": "5h", "limit": apiKey.RateLimit5h, "used": used, "remaining": max(0, apiKey.RateLimit5h-used), "window_start": rateLimitData.Window5hStart}
				if rateLimitData.Window5hStart != nil && !service.IsWindowExpired(rateLimitData.Window5hStart, service.RateLimitWindow5h) {
					entry["reset_at"] = rateLimitData.Window5hStart.Add(service.RateLimitWindow5h)
				}
				rateLimits = append(rateLimits, entry)
			}
			if apiKey.RateLimit1d > 0 {
				used := rateLimitData.EffectiveUsage1d()
				entry := gin.H{"window": "1d", "limit": apiKey.RateLimit1d, "used": used, "remaining": max(0, apiKey.RateLimit1d-used), "window_start": rateLimitData.Window1dStart}
				if rateLimitData.Window1dStart != nil && !service.IsWindowExpired(rateLimitData.Window1dStart, service.RateLimitWindow1d) {
					entry["reset_at"] = rateLimitData.Window1dStart.Add(service.RateLimitWindow1d)
				}
				rateLimits = append(rateLimits, entry)
			}
			if apiKey.RateLimit7d > 0 {
				used := rateLimitData.EffectiveUsage7d()
				entry := gin.H{"window": "7d", "limit": apiKey.RateLimit7d, "used": used, "remaining": max(0, apiKey.RateLimit7d-used), "window_start": rateLimitData.Window7dStart}
				if rateLimitData.Window7dStart != nil && !service.IsWindowExpired(rateLimitData.Window7dStart, service.RateLimitWindow7d) {
					entry["reset_at"] = rateLimitData.Window7dStart.Add(service.RateLimitWindow7d)
				}
				rateLimits = append(rateLimits, entry)
			}
			if len(rateLimits) > 0 {
				resp["rate_limits"] = rateLimits
			}
		}
	}
	if apiKey.ExpiresAt != nil {
		resp["expires_at"] = apiKey.ExpiresAt
		resp["days_until_expiry"] = apiKey.GetDaysUntilExpiry()
	}
	if usageData != nil {
		resp["usage"] = usageData
	}
	if modelStats != nil {
		resp["model_stats"] = modelStats
	}
	c.JSON(http.StatusOK, resp)
}
func (h *GatewayHandler) usageUnrestricted(c *gin.Context, ctx context.Context, apiKey *service.APIKey, subject middleware2.AuthSubject, usageData gin.H, modelStats any) {
	if apiKey.Group != nil && apiKey.Group.IsSubscriptionType() {
		resp := gin.H{"mode": "unrestricted", "isValid": true, "planName": apiKey.Group.Name, "unit": "USD"}
		subscription, ok := middleware2.GetSubscriptionFromContext(c)
		if ok {
			remaining := h.calculateSubscriptionRemaining(apiKey.Group, subscription)
			resp["remaining"] = remaining
			resp["subscription"] = gin.H{"daily_usage_usd": subscription.DailyUsageUSD, "weekly_usage_usd": subscription.WeeklyUsageUSD, "monthly_usage_usd": subscription.MonthlyUsageUSD, "daily_limit_usd": apiKey.Group.DailyLimitUSD, "weekly_limit_usd": apiKey.Group.WeeklyLimitUSD, "monthly_limit_usd": apiKey.Group.MonthlyLimitUSD, "expires_at": subscription.ExpiresAt}
		}
		if usageData != nil {
			resp["usage"] = usageData
		}
		if modelStats != nil {
			resp["model_stats"] = modelStats
		}
		c.JSON(http.StatusOK, resp)
		return
	}
	latestUser, err := h.userService.GetByID(ctx, subject.UserID)
	if err != nil {
		h.errorResponse(c, http.StatusInternalServerError, "api_error", "Failed to get user info")
		return
	}
	resp := gin.H{"mode": "unrestricted", "isValid": true, "planName": "钱包余额", "remaining": latestUser.Balance, "unit": "USD", "balance": latestUser.Balance}
	if usageData != nil {
		resp["usage"] = usageData
	}
	if modelStats != nil {
		resp["model_stats"] = modelStats
	}
	c.JSON(http.StatusOK, resp)
}
func (h *GatewayHandler) calculateSubscriptionRemaining(group *service.Group, sub *service.UserSubscription) float64 {
	var remainingValues []float64
	if group.HasDailyLimit() {
		remaining := *group.DailyLimitUSD - sub.DailyUsageUSD
		if remaining <= 0 {
			return 0
		}
		remainingValues = append(remainingValues, remaining)
	}
	if group.HasWeeklyLimit() {
		remaining := *group.WeeklyLimitUSD - sub.WeeklyUsageUSD
		if remaining <= 0 {
			return 0
		}
		remainingValues = append(remainingValues, remaining)
	}
	if group.HasMonthlyLimit() {
		remaining := *group.MonthlyLimitUSD - sub.MonthlyUsageUSD
		if remaining <= 0 {
			return 0
		}
		remainingValues = append(remainingValues, remaining)
	}
	if len(remainingValues) == 0 {
		return -1
	}
	min := remainingValues[0]
	for _, v := range remainingValues[1:] {
		if v < min {
			min = v
		}
	}
	return min
}
