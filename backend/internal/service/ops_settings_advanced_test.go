package service

import (
	"context"
	"encoding/json"
	"testing"
)

func TestGetOpsAdvancedSettings_DefaultHidesOpenAITokenStats(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	if cfg.DisplayOpenAITokenStats {
		t.Fatalf("DisplayOpenAITokenStats = true, want false by default")
	}
	if !cfg.DisplayAlertEvents {
		t.Fatalf("DisplayAlertEvents = false, want true by default")
	}
	if repo.setCalls != 1 {
		t.Fatalf("expected defaults to be persisted once, got %d", repo.setCalls)
	}
}

func TestUpdateOpsAdvancedSettings_PersistsOpenAITokenStatsVisibility(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg := defaultOpsAdvancedSettings()
	cfg.DisplayOpenAITokenStats = true
	cfg.DisplayAlertEvents = false

	updated, err := svc.UpdateOpsAdvancedSettings(context.Background(), cfg)
	if err != nil {
		t.Fatalf("UpdateOpsAdvancedSettings() error = %v", err)
	}
	if !updated.DisplayOpenAITokenStats {
		t.Fatalf("DisplayOpenAITokenStats = false, want true")
	}
	if updated.DisplayAlertEvents {
		t.Fatalf("DisplayAlertEvents = true, want false")
	}

	reloaded, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() after update error = %v", err)
	}
	if !reloaded.DisplayOpenAITokenStats {
		t.Fatalf("reloaded DisplayOpenAITokenStats = false, want true")
	}
	if reloaded.DisplayAlertEvents {
		t.Fatalf("reloaded DisplayAlertEvents = true, want false")
	}
}

func TestGetOpsAdvancedSettings_BackfillsNewDisplayFlagsFromDefaults(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	legacyCfg := map[string]any{
		"data_retention": map[string]any{
			"cleanup_enabled":               false,
			"cleanup_schedule":              "0 2 * * *",
			"error_log_retention_days":      30,
			"minute_metrics_retention_days": 30,
			"hourly_metrics_retention_days": 30,
		},
		"aggregation": map[string]any{
			"aggregation_enabled": false,
		},
		"ignore_count_tokens_errors":    true,
		"ignore_context_canceled":       true,
		"ignore_no_available_accounts":  false,
		"ignore_invalid_api_key_errors": false,
		"auto_refresh_enabled":          false,
		"auto_refresh_interval_seconds": 30,
	}
	raw, err := json.Marshal(legacyCfg)
	if err != nil {
		t.Fatalf("marshal legacy config: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	cfg, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	if cfg.DisplayOpenAITokenStats {
		t.Fatalf("DisplayOpenAITokenStats = true, want false default backfill")
	}
	if !cfg.DisplayAlertEvents {
		t.Fatalf("DisplayAlertEvents = false, want true default backfill")
	}
}

func TestGetOpsAdvancedSettings_BackfillsRequestDetailCleanupFromDataRetentionWhenMissing(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	legacyCfg := map[string]any{
		"data_retention": map[string]any{
			"cleanup_enabled":               false,
			"cleanup_schedule":              "15 3 * * *",
			"error_log_retention_days":      30,
			"minute_metrics_retention_days": 30,
			"hourly_metrics_retention_days": 30,
		},
		"aggregation": map[string]any{
			"aggregation_enabled": false,
		},
		"request_details_enabled": true,
		// Intentionally omit request_detail_cleanup_enabled/schedule to emulate pre-field installs.
	}
	raw, err := json.Marshal(legacyCfg)
	if err != nil {
		t.Fatalf("marshal legacy config: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	cfg, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}

	if cfg.DataRetention.CleanupEnabled {
		t.Fatalf("DataRetention.CleanupEnabled = true, want false")
	}
	if cfg.RequestDetailCleanupEnabled {
		t.Fatalf("RequestDetailCleanupEnabled = true, want false inherited from data_retention.cleanup_enabled")
	}
	if cfg.RequestDetailCleanupSchedule != "15 3 * * *" {
		t.Fatalf("RequestDetailCleanupSchedule = %q, want %q inherited from data_retention.cleanup_schedule", cfg.RequestDetailCleanupSchedule, "15 3 * * *")
	}
}

func TestUpdateOpsAdvancedSettings_AllowsZeroRetentionDays(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg := defaultOpsAdvancedSettings()
	cfg.DataRetention.ErrorLogRetentionDays = 0
	cfg.DataRetention.MinuteMetricsRetentionDays = 0
	cfg.DataRetention.HourlyMetricsRetentionDays = 0
	cfg.RequestDetailRetentionDays = 0

	updated, err := svc.UpdateOpsAdvancedSettings(context.Background(), cfg)
	if err != nil {
		t.Fatalf("UpdateOpsAdvancedSettings() error = %v", err)
	}
	if updated.DataRetention.ErrorLogRetentionDays != 0 {
		t.Fatalf("ErrorLogRetentionDays = %d, want 0", updated.DataRetention.ErrorLogRetentionDays)
	}
	if updated.DataRetention.MinuteMetricsRetentionDays != 0 {
		t.Fatalf("MinuteMetricsRetentionDays = %d, want 0", updated.DataRetention.MinuteMetricsRetentionDays)
	}
	if updated.DataRetention.HourlyMetricsRetentionDays != 0 {
		t.Fatalf("HourlyMetricsRetentionDays = %d, want 0", updated.DataRetention.HourlyMetricsRetentionDays)
	}
	if updated.RequestDetailRetentionDays != 0 {
		t.Fatalf("RequestDetailRetentionDays = %d, want 0", updated.RequestDetailRetentionDays)
	}

	reloaded, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	if reloaded.DataRetention.ErrorLogRetentionDays != 0 {
		t.Fatalf("reloaded ErrorLogRetentionDays = %d, want 0", reloaded.DataRetention.ErrorLogRetentionDays)
	}
	if reloaded.RequestDetailRetentionDays != 0 {
		t.Fatalf("reloaded RequestDetailRetentionDays = %d, want 0", reloaded.RequestDetailRetentionDays)
	}
}

func TestGetOpsAdvancedSettings_DefaultPreviewLimit(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg, err := svc.GetOpsAdvancedSettings(context.Background())
	if err != nil {
		t.Fatalf("GetOpsAdvancedSettings() error = %v", err)
	}
	if cfg.RequestDetailPayloadPreviewLimitBytes != opsTracePayloadInlineBytesLimit {
		t.Fatalf("RequestDetailPayloadPreviewLimitBytes = %d, want %d", cfg.RequestDetailPayloadPreviewLimitBytes, opsTracePayloadInlineBytesLimit)
	}
}

func TestUpdateOpsAdvancedSettings_ValidatesPreviewLimitRange(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	svc := &OpsService{settingRepo: repo}

	cfg := defaultOpsAdvancedSettings()
	cfg.RequestDetailPayloadPreviewLimitBytes = 2048
	if _, err := svc.UpdateOpsAdvancedSettings(context.Background(), cfg); err == nil {
		t.Fatalf("expected preview limit validation error")
	}

	cfg.RequestDetailPayloadPreviewLimitBytes = 32768
	updated, err := svc.UpdateOpsAdvancedSettings(context.Background(), cfg)
	if err != nil {
		t.Fatalf("UpdateOpsAdvancedSettings() error = %v", err)
	}
	if updated.RequestDetailPayloadPreviewLimitBytes != 32768 {
		t.Fatalf("RequestDetailPayloadPreviewLimitBytes = %d, want 32768", updated.RequestDetailPayloadPreviewLimitBytes)
	}
}
