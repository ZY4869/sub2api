package service

import (
	"context"
	"log/slog"
	"time"
)

func (s *RateLimitService) PreCheckUsage(ctx context.Context, account *Account, requestedModel string) (bool, error) {
	if account == nil || EffectiveProtocol(account) != PlatformGemini {
		return true, nil
	}
	if s.usageRepo == nil || s.geminiQuotaService == nil {
		return true, nil
	}

	quota, ok := s.geminiQuotaService.QuotaForAccount(ctx, account)
	if !ok {
		return true, nil
	}

	now := time.Now()
	modelClass := geminiModelClassFromName(requestedModel)

	// 1) Daily quota precheck (RPD; resets at PST midnight)
	{
		var limit int64
		if quota.SharedRPD > 0 {
			limit = quota.SharedRPD
		} else {
			switch modelClass {
			case geminiModelFlash:
				limit = quota.FlashRPD
			default:
				limit = quota.ProRPD
			}
		}

		if limit > 0 {
			start := geminiDailyWindowStart(now)
			totals, ok := s.getGeminiUsageTotals(account.ID, start, now)
			if !ok {
				stats, err := s.usageRepo.GetModelStatsWithFilters(ctx, start, now, 0, 0, account.ID, 0, 0, nil, nil, nil)
				if err != nil {
					return true, err
				}
				totals = geminiAggregateUsage(stats)
				s.setGeminiUsageTotals(account.ID, start, now, totals)
			}

			var used int64
			if quota.SharedRPD > 0 {
				used = totals.ProRequests + totals.FlashRequests
			} else {
				switch modelClass {
				case geminiModelFlash:
					used = totals.FlashRequests
				default:
					used = totals.ProRequests
				}
			}

			if used >= limit {
				resetAt := geminiDailyResetTime(now)
				// NOTE:
				// - This is a local precheck to reduce upstream 429s.
				// - Do NOT mark the account as rate-limited here; rate_limit_reset_at should reflect real upstream 429s.
				slog.Info("gemini_precheck_daily_quota_reached", "account_id", account.ID, "used", used, "limit", limit, "reset_at", resetAt)
				return false, nil
			}
		}
	}

	// 2) Minute quota precheck (RPM; fixed window current minute)
	{
		var limit int64
		if quota.SharedRPM > 0 {
			limit = quota.SharedRPM
		} else {
			switch modelClass {
			case geminiModelFlash:
				limit = quota.FlashRPM
			default:
				limit = quota.ProRPM
			}
		}

		if limit > 0 {
			start := now.Truncate(time.Minute)
			stats, err := s.usageRepo.GetModelStatsWithFilters(ctx, start, now, 0, 0, account.ID, 0, 0, nil, nil, nil)
			if err != nil {
				return true, err
			}
			totals := geminiAggregateUsage(stats)

			var used int64
			if quota.SharedRPM > 0 {
				used = totals.ProRequests + totals.FlashRequests
			} else {
				switch modelClass {
				case geminiModelFlash:
					used = totals.FlashRequests
				default:
					used = totals.ProRequests
				}
			}

			if used >= limit {
				resetAt := start.Add(time.Minute)
				// Do not persist "rate limited" status from local precheck. See note above.
				slog.Info("gemini_precheck_minute_quota_reached", "account_id", account.ID, "used", used, "limit", limit, "reset_at", resetAt)
				return false, nil
			}
		}
	}

	return true, nil
}
