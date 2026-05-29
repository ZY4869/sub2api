package service

import (
	"context"
	"log/slog"
	"time"
)

// handle529 处理529过载错误
// 根据配置决定是否暂停账号调度及冷却时长
func (s *RateLimitService) handle529(ctx context.Context, account *Account) {
	var settings *OverloadCooldownSettings
	if s.settingService != nil {
		var err error
		settings, err = s.settingService.GetOverloadCooldownSettings(ctx)
		if err != nil {
			slog.Warn("overload_settings_read_failed", "account_id", account.ID, "error", err)
			settings = nil
		}
	}
	// 回退到配置文件
	if settings == nil {
		cooldown := s.cfg.RateLimit.OverloadCooldownMinutes
		if cooldown <= 0 {
			cooldown = 10
		}
		settings = &OverloadCooldownSettings{Enabled: true, CooldownMinutes: cooldown}
	}

	if !settings.Enabled {
		slog.Info("account_529_ignored", "account_id", account.ID, "reason", "overload_cooldown_disabled")
		return
	}

	cooldownMinutes := settings.CooldownMinutes
	if cooldownMinutes <= 0 {
		cooldownMinutes = 10
	}

	until := time.Now().Add(time.Duration(cooldownMinutes) * time.Minute)
	if err := s.accountRepo.SetOverloaded(ctx, account.ID, until); err != nil {
		slog.Warn("overload_set_failed", "account_id", account.ID, "error", err)
		return
	}

	slog.Info("account_overloaded", "account_id", account.ID, "until", until)
}
