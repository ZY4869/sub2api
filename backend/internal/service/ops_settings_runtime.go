package service

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func defaultOpsAdvancedSettingsWithConfig(cfg *config.Config) *OpsAdvancedSettings {
	settings := defaultOpsAdvancedSettings()
	if cfg == nil {
		return settings
	}

	settings.DataRetention.CleanupEnabled = cfg.Ops.Cleanup.Enabled
	if value := strings.TrimSpace(cfg.Ops.Cleanup.Schedule); value != "" {
		settings.DataRetention.CleanupSchedule = value
	}
	if days := cfg.Ops.Cleanup.ErrorLogRetentionDays; days > 0 {
		settings.DataRetention.ErrorLogRetentionDays = days
	}
	if days := cfg.Ops.Cleanup.MinuteMetricsRetentionDays; days > 0 {
		settings.DataRetention.MinuteMetricsRetentionDays = days
	}
	if days := cfg.Ops.Cleanup.HourlyMetricsRetentionDays; days > 0 {
		settings.DataRetention.HourlyMetricsRetentionDays = days
	}

	settings.RequestDetailsEnabled = cfg.Ops.RequestDetails.Enabled
	settings.RequestDetailCleanupEnabled = cfg.Ops.Cleanup.Enabled
	settings.RequestDetailCleanupSchedule = settings.DataRetention.CleanupSchedule
	if days := cfg.Ops.RequestDetails.RetentionDays; days > 0 {
		settings.RequestDetailRetentionDays = days
	}
	if cfg.Ops.RequestDetails.SuccessSampleRate >= 0 {
		settings.SuccessSampleRate = cfg.Ops.RequestDetails.SuccessSampleRate
	}
	if value := cfg.Ops.RequestDetails.ForceCaptureSlowMs; value > 0 {
		settings.ForceCaptureSlowMs = value
	}
	if value := cfg.Ops.RequestDetails.RawExportMaxRows; value > 0 {
		settings.RawExportMaxRows = value
	}
	return settings
}

func loadOpsAdvancedSettings(ctx context.Context, repo SettingRepository, cfg *config.Config) (*OpsAdvancedSettings, error) {
	if repo == nil {
		return defaultOpsAdvancedSettingsWithConfig(cfg), nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	raw, err := repo.GetValue(ctx, SettingKeyOpsAdvancedSettings)
	if err != nil || strings.TrimSpace(raw) == "" {
		return defaultOpsAdvancedSettingsWithConfig(cfg), err
	}

	settings := defaultOpsAdvancedSettingsWithConfig(cfg)
	if err := json.Unmarshal([]byte(raw), settings); err != nil {
		return defaultOpsAdvancedSettingsWithConfig(cfg), err
	}

	backfillOpsAdvancedSettingsRequestDetailCleanup(raw, settings)
	normalizeOpsAdvancedSettings(settings)
	return settings, nil
}
