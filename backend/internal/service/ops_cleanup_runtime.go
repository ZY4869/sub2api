package service

import (
	"context"
	"encoding/json"
	"strings"
)

type opsCleanupRetentionConfig struct {
	ErrorLogRetentionDays      int
	RequestTraceRetentionDays  int
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

func (s *OpsCleanupService) cleanupRetentionConfig(ctx context.Context) opsCleanupRetentionConfig {
	out := opsCleanupRetentionConfig{
		ErrorLogRetentionDays:      30,
		RequestTraceRetentionDays:  30,
		MinuteMetricsRetentionDays: 30,
		HourlyMetricsRetentionDays: 30,
	}
	if s != nil && s.cfg != nil {
		if days := s.cfg.Ops.Cleanup.ErrorLogRetentionDays; days > 0 {
			out.ErrorLogRetentionDays = days
		}
		if days := s.cfg.Ops.RequestDetails.RetentionDays; days > 0 {
			out.RequestTraceRetentionDays = days
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
		if days := advanced.RequestDetailRetentionDays; days > 0 {
			out.RequestTraceRetentionDays = days
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

func (s *OpsCleanupService) loadCleanupAdvancedSettings(ctx context.Context) *OpsAdvancedSettings {
	if s == nil || s.settingRepo == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyOpsAdvancedSettings)
	if err != nil || strings.TrimSpace(raw) == "" {
		return nil
	}
	cfg := defaultOpsAdvancedSettings()
	if err := json.Unmarshal([]byte(raw), cfg); err != nil {
		return nil
	}
	normalizeOpsAdvancedSettings(cfg)
	return cfg
}
