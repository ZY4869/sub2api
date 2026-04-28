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
	if retention.RequestTraceRetentionDays != 7 {
		t.Fatalf("RequestTraceRetentionDays = %d, want 7", retention.RequestTraceRetentionDays)
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
