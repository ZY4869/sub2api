package service

import (
	"testing"
	"time"
)

func TestAccountDisplayRateLimitState(t *testing.T) {
	t.Parallel()

	now := time.Date(2026, 4, 22, 10, 0, 0, 0, time.UTC)
	fiveHourReset := now.Add(2 * time.Hour).UTC().Truncate(time.Second)
	sevenDayReset := now.Add(24 * time.Hour).UTC().Truncate(time.Second)
	laterSevenDayReset := now.Add(48 * time.Hour).UTC().Truncate(time.Second)

	tests := []struct {
		name       string
		account    *Account
		wantLimit  bool
		wantReason string
		wantReset  *time.Time
	}{
		{
			name: "non pro 5h uses single scope",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					"codex_5h_used_percent": 100.0,
					"codex_5h_reset_at":     fiveHourReset.Format(time.RFC3339),
				},
			},
			wantLimit:  true,
			wantReason: AccountRateLimitReasonUsage5h,
			wantReset:  &fiveHourReset,
		},
		{
			name: "non pro 7d uses single scope",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Extra: map[string]any{
					"codex_7d_used_percent": 100.0,
					"codex_7d_reset_at":     sevenDayReset.Format(time.RFC3339),
				},
			},
			wantLimit:  true,
			wantReason: AccountRateLimitReasonUsage7d,
			wantReset:  &sevenDayReset,
		},
		{
			name: "pro normal only does not become whole account limit",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"codex_7d_used_percent": 100.0,
					"codex_7d_reset_at":     sevenDayReset.Format(time.RFC3339),
				},
			},
			wantLimit: false,
		},
		{
			name: "pro spark only does not become whole account limit",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					codexSpark7dUsedPercentKey: 100.0,
					codexSpark7dResetAtKey:     sevenDayReset.Format(time.RFC3339),
				},
			},
			wantLimit: false,
		},
		{
			name: "pro both 5h uses earlier reset",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"codex_5h_used_percent":    100.0,
					"codex_5h_reset_at":        laterSevenDayReset.Format(time.RFC3339),
					codexSpark5hUsedPercentKey: 100.0,
					codexSpark5hResetAtKey:     fiveHourReset.Format(time.RFC3339),
				},
			},
			wantLimit:  true,
			wantReason: AccountRateLimitReasonUsage5h,
			wantReset:  &fiveHourReset,
		},
		{
			name: "pro mixed 5h and 7d uses usage 7d with earlier reset",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"codex_7d_used_percent":    100.0,
					"codex_7d_reset_at":        laterSevenDayReset.Format(time.RFC3339),
					codexSpark5hUsedPercentKey: 100.0,
					codexSpark5hResetAtKey:     fiveHourReset.Format(time.RFC3339),
				},
			},
			wantLimit:  true,
			wantReason: AccountRateLimitReasonUsage7d,
			wantReset:  &fiveHourReset,
		},
		{
			name: "pro both 7d uses usage 7d all and later reset",
			account: &Account{
				Platform: PlatformOpenAI,
				Type:     AccountTypeOAuth,
				Credentials: map[string]any{
					"plan_type": "pro",
				},
				Extra: map[string]any{
					"codex_7d_used_percent":    100.0,
					"codex_7d_reset_at":        sevenDayReset.Format(time.RFC3339),
					codexSpark7dUsedPercentKey: 100.0,
					codexSpark7dResetAtKey:     laterSevenDayReset.Format(time.RFC3339),
				},
			},
			wantLimit:  true,
			wantReason: AccountRateLimitReasonUsage7dAll,
			wantReset:  &laterSevenDayReset,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := AccountDisplayRateLimitState(tt.account, now)
			if state.Limited != tt.wantLimit {
				t.Fatalf("Limited = %v, want %v", state.Limited, tt.wantLimit)
			}
			if state.Reason != tt.wantReason {
				t.Fatalf("Reason = %q, want %q", state.Reason, tt.wantReason)
			}
			switch {
			case tt.wantReset == nil:
				if state.ResetAt != nil {
					t.Fatalf("ResetAt = %v, want nil", state.ResetAt)
				}
			case state.ResetAt == nil:
				t.Fatalf("ResetAt = nil, want %v", tt.wantReset)
			case !state.ResetAt.Equal(*tt.wantReset):
				t.Fatalf("ResetAt = %v, want %v", *state.ResetAt, *tt.wantReset)
			}
		})
	}
}
