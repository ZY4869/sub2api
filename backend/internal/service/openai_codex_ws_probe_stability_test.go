package service

import (
	"context"
	"testing"
	"time"
)

func TestAccountUsageService_ProbeOpenAICodexSnapshot_Pro_WSEventAfterNonQuotaEventsDistinguishesScopes(t *testing.T) {
	t.Parallel()

	normalConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created"}`),
			[]byte(`{"type":"response.in_progress"}`),
			[]byte(`{"type":"codex.rate_limits","rate_limits":{"primary":{"used_percent":4.0,"window_minutes":300,"resets_in_seconds":1800},"secondary":{"used_percent":40.0,"window_minutes":10080,"resets_in_seconds":604800}}}`),
			[]byte(`{"type":"response.completed","response":{"id":"resp_normal_1","model":"gpt-5.3-codex","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}
	sparkConn := &openAIWSCaptureConn{
		events: [][]byte{
			[]byte(`{"type":"response.created"}`),
			[]byte(`{"type":"response.in_progress"}`),
			[]byte(`{"type":"codex.rate_limits","rate_limits":{"primary":{"used_percent":7.0,"window_minutes":300,"resets_in_seconds":1200},"secondary":{"used_percent":55.0,"window_minutes":10080,"resets_in_seconds":500000}}}`),
			[]byte(`{"type":"response.completed","response":{"id":"resp_spark_1","model":"gpt-5.3-codex-spark","usage":{"input_tokens":1,"output_tokens":1}}}`),
		},
	}

	dialer := &openAICodexProbeQueueDialer{conns: []openAIWSClientConn{normalConn, sparkConn}}
	svc := &AccountUsageService{
		openAICodexWSProbeDialer:              dialer,
		openAICodexWSProbeReadTimeoutOverride: 200 * time.Millisecond,
	}

	updates, _, err := svc.probeOpenAICodexSnapshot(context.Background(), &Account{
		ID:       9104,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token": "token",
			"plan_type":    "pro",
		},
	})
	if err != nil {
		t.Fatalf("probeOpenAICodexSnapshot() error = %v", err)
	}
	if got := parseExtraFloat64(updates["codex_5h_used_percent"]); got != 4.0 {
		t.Fatalf("codex_5h_used_percent = %v, want 4", got)
	}
	if got := parseExtraFloat64(updates[codexSpark5hUsedPercentKey]); got != 7.0 {
		t.Fatalf("codex_spark_5h_used_percent = %v, want 7", got)
	}
	if got := parseExtraFloat64(updates["codex_7d_used_percent"]); got != 40.0 {
		t.Fatalf("codex_7d_used_percent = %v, want 40", got)
	}
	if got := parseExtraFloat64(updates[codexSpark7dUsedPercentKey]); got != 55.0 {
		t.Fatalf("codex_spark_7d_used_percent = %v, want 55", got)
	}
}
