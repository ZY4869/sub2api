package service

import (
	"testing"
	"time"
)

func TestParseCodexRateLimitsFromWSMessage_PayloadNestedRateLimits(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 30, 0, 0, 0, 0, time.UTC)
	msg := []byte(`{"type":"codex.rate_limits","payload":{"rate_limits":{"rate_limits":{"primary":{"used_percent":12.5,"window_minutes":300,"resets_in_seconds":1800},"secondary":{"used_percent":88.0,"window_minutes":10080,"resets_in_seconds":604800}}}}}`)
	snapshot := parseCodexRateLimitsFromWSMessage(msg, now)
	if snapshot == nil {
		t.Fatal("expected snapshot to be parsed from nested payload.rate_limits.rate_limits")
	}
	if snapshot.PrimaryUsedPercent == nil || *snapshot.PrimaryUsedPercent != 12.5 {
		t.Fatalf("PrimaryUsedPercent = %v, want 12.5", snapshot.PrimaryUsedPercent)
	}
	if snapshot.PrimaryWindowMinutes == nil || *snapshot.PrimaryWindowMinutes != 300 {
		t.Fatalf("PrimaryWindowMinutes = %v, want 300", snapshot.PrimaryWindowMinutes)
	}
	if snapshot.PrimaryResetAfterSeconds == nil || *snapshot.PrimaryResetAfterSeconds != 1800 {
		t.Fatalf("PrimaryResetAfterSeconds = %v, want 1800", snapshot.PrimaryResetAfterSeconds)
	}
	if snapshot.SecondaryUsedPercent == nil || *snapshot.SecondaryUsedPercent != 88.0 {
		t.Fatalf("SecondaryUsedPercent = %v, want 88", snapshot.SecondaryUsedPercent)
	}
	if snapshot.SecondaryWindowMinutes == nil || *snapshot.SecondaryWindowMinutes != 10080 {
		t.Fatalf("SecondaryWindowMinutes = %v, want 10080", snapshot.SecondaryWindowMinutes)
	}
	if snapshot.SecondaryResetAfterSeconds == nil || *snapshot.SecondaryResetAfterSeconds != 604800 {
		t.Fatalf("SecondaryResetAfterSeconds = %v, want 604800", snapshot.SecondaryResetAfterSeconds)
	}
}
