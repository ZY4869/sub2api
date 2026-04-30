package service

import (
	"context"
	"strings"
)

type opsCleanupRetentionConfig struct {
	ErrorLogRetentionDays      int
	MinuteMetricsRetentionDays int
	HourlyMetricsRetentionDays int
}

func (s *OpsCleanupService) cleanupSchedule(ctx context.Context) string {
	schedule := "0 2 * * *"
	if s != nil && s.cfg != nil && strings.TrimSpace(s.cfg.Ops.Cleanup.Schedule) != "" {
		schedule = strings.TrimSpace(s.cfg.Ops.Cleanup.Schedule)
	}
	if advanced := s.loadCleanupAdvancedSettings(ctx); advanced != nil {
		if value := strings.TrimSpace(advanced.DataRetention.CleanupSchedule); value != "" {
			schedule = value
		}
	}
	return schedule
}

func (s *OpsCleanupService) cleanupEnabled(ctx context.Context) bool {
	enabled := true
	if s != nil && s.cfg != nil {
		enabled = s.cfg.Ops.Cleanup.Enabled
	}
	if advanced := s.loadCleanupAdvancedSettings(ctx); advanced != nil {
		enabled = advanced.DataRetention.CleanupEnabled
	}
	return enabled
}

func (s *OpsCleanupService) requestDetailCleanupEnabled(ctx context.Context) bool {
	enabled := true
	if s != nil && s.cfg != nil {
		enabled = s.cfg.Ops.Cleanup.Enabled
	}
	if advanced := s.loadCleanupAdvancedSettings(ctx); advanced != nil {
		enabled = advanced.RequestDetailCleanupEnabled
	}
	return enabled
}

func (s *OpsCleanupService) requestDetailCleanupSchedule(ctx context.Context) string {
	schedule := s.cleanupSchedule(ctx)
	if advanced := s.loadCleanupAdvancedSettings(ctx); advanced != nil {
		if value := strings.TrimSpace(advanced.RequestDetailCleanupSchedule); value != "" {
			schedule = value
		}
	}
	return schedule
}

func (s *OpsCleanupService) cleanupRetentionConfig(ctx context.Context) opsCleanupRetentionConfig {
	out := opsCleanupRetentionConfig{
		ErrorLogRetentionDays:      30,
		MinuteMetricsRetentionDays: 30,
		HourlyMetricsRetentionDays: 30,
	}
	if s != nil && s.cfg != nil {
		if days := s.cfg.Ops.Cleanup.ErrorLogRetentionDays; days > 0 {
			out.ErrorLogRetentionDays = days
		}
		if days := s.cfg.Ops.Cleanup.MinuteMetricsRetentionDays; days > 0 {
			out.MinuteMetricsRetentionDays = days
		}
		if days := s.cfg.Ops.Cleanup.HourlyMetricsRetentionDays; days > 0 {
			out.HourlyMetricsRetentionDays = days
		}
	}
	if advanced := s.loadCleanupAdvancedSettings(ctx); advanced != nil {
		if days := advanced.DataRetention.ErrorLogRetentionDays; days > 0 {
			out.ErrorLogRetentionDays = days
		}
		if days := advanced.DataRetention.MinuteMetricsRetentionDays; days > 0 {
			out.MinuteMetricsRetentionDays = days
		}
		if days := advanced.DataRetention.HourlyMetricsRetentionDays; days > 0 {
			out.HourlyMetricsRetentionDays = days
		}
	}
	return out
}

func (s *OpsCleanupService) requestDetailRetentionDays(ctx context.Context) int {
	days := 30
	if s != nil && s.cfg != nil && s.cfg.Ops.RequestDetails.RetentionDays > 0 {
		days = s.cfg.Ops.RequestDetails.RetentionDays
	}
	if advanced := s.loadCleanupAdvancedSettings(ctx); advanced != nil && advanced.RequestDetailRetentionDays > 0 {
		days = advanced.RequestDetailRetentionDays
	}
	return days
}

func (s *OpsCleanupService) loadCleanupAdvancedSettings(ctx context.Context) *OpsAdvancedSettings {
	if s == nil {
		return nil
	}
	cfg, err := loadOpsAdvancedSettings(ctx, s.settingRepo, s.cfg)
	if err != nil {
		return nil
	}
	return cfg
}
