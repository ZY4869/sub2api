package service

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func TestOpsCleanupRetentionConfigUsesAdvancedRequestTraceRetention(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	advanced := defaultOpsAdvancedSettings()
	advanced.DataRetention.ErrorLogRetentionDays = 14
	advanced.RequestDetailRetentionDays = 7
	raw, err := json.Marshal(advanced)
	if err != nil {
		t.Fatalf("marshal advanced settings: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	svc := NewOpsCleanupService(nil, repo, nil, nil, &config.Config{
		Ops: config.OpsConfig{
			Cleanup: config.OpsCleanupConfig{
				ErrorLogRetentionDays:      30,
				MinuteMetricsRetentionDays: 30,
				HourlyMetricsRetentionDays: 30,
			},
			RequestDetails: config.OpsRequestDetailsConfig{
				RetentionDays: 30,
			},
		},
	})

	retention := svc.cleanupRetentionConfig(context.Background())
	if retention.ErrorLogRetentionDays != 14 {
		t.Fatalf("ErrorLogRetentionDays = %d, want 14", retention.ErrorLogRetentionDays)
	}
	if got := svc.requestDetailRetentionDays(context.Background()); got != 7 {
		t.Fatalf("requestDetailRetentionDays() = %d, want 7", got)
	}
}

func TestOpsCleanupScheduleUsesAdvancedSetting(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	advanced := defaultOpsAdvancedSettings()
	advanced.DataRetention.CleanupSchedule = "*/15 * * * *"
	raw, err := json.Marshal(advanced)
	if err != nil {
		t.Fatalf("marshal advanced settings: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	svc := NewOpsCleanupService(nil, repo, nil, nil, &config.Config{
		Ops: config.OpsConfig{
			Cleanup: config.OpsCleanupConfig{Schedule: "0 2 * * *"},
		},
	})

	if got := svc.cleanupSchedule(context.Background()); got != "*/15 * * * *" {
		t.Fatalf("cleanupSchedule() = %q, want advanced schedule", got)
	}
}

func TestOpsCleanupUsesIndependentRequestDetailCleanupConfig(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	advanced := defaultOpsAdvancedSettings()
	advanced.DataRetention.CleanupEnabled = false
	advanced.DataRetention.CleanupSchedule = "0 1 * * *"
	advanced.RequestDetailCleanupEnabled = true
	advanced.RequestDetailCleanupSchedule = "0 5 * * *"
	raw, err := json.Marshal(advanced)
	if err != nil {
		t.Fatalf("marshal advanced settings: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	svc := NewOpsCleanupService(nil, repo, nil, nil, &config.Config{
		Ops: config.OpsConfig{
			Cleanup: config.OpsCleanupConfig{
				Enabled:  true,
				Schedule: "0 2 * * *",
			},
		},
	})

	if got := svc.cleanupEnabled(context.Background()); got {
		t.Fatalf("cleanupEnabled() = true, want false (advanced override)")
	}
	if got := svc.requestDetailCleanupEnabled(context.Background()); !got {
		t.Fatalf("requestDetailCleanupEnabled() = false, want true (advanced override)")
	}
	if got := svc.cleanupSchedule(context.Background()); got != "0 1 * * *" {
		t.Fatalf("cleanupSchedule() = %q, want %q", got, "0 1 * * *")
	}
	if got := svc.requestDetailCleanupSchedule(context.Background()); got != "0 5 * * *" {
		t.Fatalf("requestDetailCleanupSchedule() = %q, want %q", got, "0 5 * * *")
	}
}

func TestOpsCleanupBackfillsRequestDetailCleanupFromDataRetentionWhenMissing(t *testing.T) {
	repo := newRuntimeSettingRepoStub()

	legacyCfg := map[string]any{
		"data_retention": map[string]any{
			"cleanup_enabled":               false,
			"cleanup_schedule":              "5 4 * * *",
			"error_log_retention_days":      30,
			"minute_metrics_retention_days": 30,
			"hourly_metrics_retention_days": 30,
		},
		"aggregation": map[string]any{
			"aggregation_enabled": false,
		},
		// Intentionally omit request_detail_cleanup_enabled/schedule.
	}
	raw, err := json.Marshal(legacyCfg)
	if err != nil {
		t.Fatalf("marshal legacy config: %v", err)
	}
	repo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	svc := NewOpsCleanupService(nil, repo, nil, nil, &config.Config{
		Ops: config.OpsConfig{
			Cleanup: config.OpsCleanupConfig{
				Enabled:  true,
				Schedule: "0 2 * * *",
			},
		},
	})

	if got := svc.requestDetailCleanupEnabled(context.Background()); got {
		t.Fatalf("requestDetailCleanupEnabled() = true, want false inherited from data_retention.cleanup_enabled")
	}
	if got := svc.requestDetailCleanupSchedule(context.Background()); got != "5 4 * * *" {
		t.Fatalf("requestDetailCleanupSchedule() = %q, want %q inherited from data_retention.cleanup_schedule", got, "5 4 * * *")
	}
}
