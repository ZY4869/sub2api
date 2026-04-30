package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

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

func TestOpsCleanupRetentionConfig_AllowsZeroRetentionDays(t *testing.T) {
	repo := newRuntimeSettingRepoStub()
	advanced := defaultOpsAdvancedSettings()
	advanced.DataRetention.ErrorLogRetentionDays = 0
	advanced.DataRetention.MinuteMetricsRetentionDays = 0
	advanced.DataRetention.HourlyMetricsRetentionDays = 0
	advanced.RequestDetailRetentionDays = 0
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
	if retention.ErrorLogRetentionDays != 0 {
		t.Fatalf("ErrorLogRetentionDays = %d, want 0", retention.ErrorLogRetentionDays)
	}
	if retention.MinuteMetricsRetentionDays != 0 {
		t.Fatalf("MinuteMetricsRetentionDays = %d, want 0", retention.MinuteMetricsRetentionDays)
	}
	if retention.HourlyMetricsRetentionDays != 0 {
		t.Fatalf("HourlyMetricsRetentionDays = %d, want 0", retention.HourlyMetricsRetentionDays)
	}
	if got := svc.requestDetailRetentionDays(context.Background()); got != 0 {
		t.Fatalf("requestDetailRetentionDays() = %d, want 0", got)
	}
}

func TestOpsCleanupRequestDetailCleanupOnce_WipesOnZeroRetentionDays(t *testing.T) {
	settingRepo := newRuntimeSettingRepoStub()
	advanced := defaultOpsAdvancedSettings()
	advanced.RequestDetailRetentionDays = 0
	raw, err := json.Marshal(advanced)
	if err != nil {
		t.Fatalf("marshal advanced settings: %v", err)
	}
	settingRepo.values[SettingKeyOpsAdvancedSettings] = string(raw)

	var called bool
	var gotCutoff time.Time
	opsRepo := &opsRepoMock{
		DeleteExpiredRequestTracesFn: func(ctx context.Context, cutoff time.Time, batchSize int) (OpsRequestTraceDeleteCounts, error) {
			called = true
			gotCutoff = cutoff
			return OpsRequestTraceDeleteCounts{DeletedTraces: 3, DeletedAudits: 4}, nil
		},
	}

	svc := NewOpsCleanupService(opsRepo, settingRepo, nil, nil, &config.Config{
		Ops: config.OpsConfig{
			Enabled: true,
			Cleanup: config.OpsCleanupConfig{
				Enabled: true,
			},
			RequestDetails: config.OpsRequestDetailsConfig{
				RetentionDays: 30,
			},
		},
	})

	startedAt := time.Now().UTC()
	counts, err := svc.runRequestDetailCleanupOnce(context.Background())
	if err != nil {
		t.Fatalf("runRequestDetailCleanupOnce() error = %v", err)
	}
	if !called {
		t.Fatalf("expected DeleteExpiredRequestTraces to be called")
	}
	// Cutoff should be far in the future to wipe all rows.
	if !gotCutoff.After(startedAt.AddDate(10, 0, 0)) {
		t.Fatalf("cutoff = %s, want > %s", gotCutoff.UTC().Format(time.RFC3339Nano), startedAt.AddDate(10, 0, 0).UTC().Format(time.RFC3339Nano))
	}
	if counts.requestTraces != 3 || counts.traceAudits != 4 {
		t.Fatalf("deleted counts = traces=%d audits=%d, want traces=3 audits=4", counts.requestTraces, counts.traceAudits)
	}
}
