package service

import (
	"strings"
	"time"
)

const (
	accountAutoRecoveryProbeCheckedAtKey = "auto_recovery_probe_checked_at"
	accountAutoRecoveryProbeStatusKey    = "auto_recovery_probe_status"
	accountAutoRecoveryProbeSummaryKey   = "auto_recovery_probe_summary"
	accountAutoRecoveryProbeBlacklisted  = "auto_recovery_probe_blacklisted"
	accountAutoRecoveryProbeNextRetryKey = "auto_recovery_probe_next_retry_at"
	accountAutoRecoveryProbeErrorCodeKey = "auto_recovery_probe_error_code"

	AccountAutoRecoveryProbeStatusSuccess        = "success"
	AccountAutoRecoveryProbeStatusRetryScheduled = "retry_scheduled"
	AccountAutoRecoveryProbeStatusBlacklisted    = "blacklisted"
)

type AccountAutoRecoveryProbeSummary struct {
	CheckedAt   string `json:"checked_at,omitempty"`
	Status      string `json:"status,omitempty"`
	Summary     string `json:"summary,omitempty"`
	Blacklisted bool   `json:"blacklisted,omitempty"`
	NextRetryAt string `json:"next_retry_at,omitempty"`
	ErrorCode   string `json:"error_code,omitempty"`
}

func AccountAutoRecoveryProbeSummaryFromExtra(extra map[string]any) *AccountAutoRecoveryProbeSummary {
	summary := &AccountAutoRecoveryProbeSummary{
		CheckedAt:   stringValueFromAny(extra[accountAutoRecoveryProbeCheckedAtKey]),
		Status:      strings.TrimSpace(stringValueFromAny(extra[accountAutoRecoveryProbeStatusKey])),
		Summary:     strings.TrimSpace(stringValueFromAny(extra[accountAutoRecoveryProbeSummaryKey])),
		Blacklisted: parseExtraBool(extra[accountAutoRecoveryProbeBlacklisted]),
		NextRetryAt: stringValueFromAny(extra[accountAutoRecoveryProbeNextRetryKey]),
		ErrorCode:   strings.TrimSpace(stringValueFromAny(extra[accountAutoRecoveryProbeErrorCodeKey])),
	}
	if summary.CheckedAt == "" && summary.Status == "" && summary.Summary == "" && summary.NextRetryAt == "" && summary.ErrorCode == "" && !summary.Blacklisted {
		return nil
	}
	return summary
}

func BuildAccountAutoRecoveryProbeExtra(
	checkedAt time.Time,
	status string,
	summary string,
	blacklisted bool,
	nextRetryAt *time.Time,
	errorCode string,
) map[string]any {
	out := map[string]any{
		accountAutoRecoveryProbeCheckedAtKey: checkedAt.UTC().Format(time.RFC3339),
		accountAutoRecoveryProbeStatusKey:    strings.TrimSpace(status),
		accountAutoRecoveryProbeSummaryKey:   strings.TrimSpace(summary),
		accountAutoRecoveryProbeBlacklisted:  blacklisted,
		accountAutoRecoveryProbeErrorCodeKey: strings.TrimSpace(errorCode),
	}
	if nextRetryAt != nil && !nextRetryAt.IsZero() {
		out[accountAutoRecoveryProbeNextRetryKey] = nextRetryAt.UTC().Format(time.RFC3339)
	} else {
		out[accountAutoRecoveryProbeNextRetryKey] = nil
	}
	if strings.TrimSpace(errorCode) == "" {
		out[accountAutoRecoveryProbeErrorCodeKey] = nil
	}
	return out
}

func parseAccountAutoRecoveryProbeTime(extra map[string]any, key string) *time.Time {
	value := strings.TrimSpace(stringValueFromAny(extra[key]))
	if value == "" {
		return nil
	}
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339} {
		if parsed, err := time.Parse(layout, value); err == nil {
			parsed = parsed.UTC()
			return &parsed
		}
	}
	return nil
}
