package service

import (
	"log/slog"
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func codexProbeSnapshotLogFields(snapshot *OpenAICodexUsageSnapshot) (float64, float64, int, int, int, int) {
	if snapshot == nil {
		return 0, 0, 0, 0, 0, 0
	}
	primaryUsed := 0.0
	secondaryUsed := 0.0
	primaryWindow := 0
	secondaryWindow := 0
	primaryResetAfter := 0
	secondaryResetAfter := 0
	if snapshot.PrimaryUsedPercent != nil {
		primaryUsed = *snapshot.PrimaryUsedPercent
	}
	if snapshot.SecondaryUsedPercent != nil {
		secondaryUsed = *snapshot.SecondaryUsedPercent
	}
	if snapshot.PrimaryWindowMinutes != nil {
		primaryWindow = *snapshot.PrimaryWindowMinutes
	}
	if snapshot.SecondaryWindowMinutes != nil {
		secondaryWindow = *snapshot.SecondaryWindowMinutes
	}
	if snapshot.PrimaryResetAfterSeconds != nil {
		primaryResetAfter = *snapshot.PrimaryResetAfterSeconds
	}
	if snapshot.SecondaryResetAfterSeconds != nil {
		secondaryResetAfter = *snapshot.SecondaryResetAfterSeconds
	}
	return primaryUsed, secondaryUsed, primaryWindow, secondaryWindow, primaryResetAfter, secondaryResetAfter
}

func codexProbeHeadersLogFields(headers http.Header) (float64, float64, int, int, int, int) {
	if headers == nil {
		return 0, 0, 0, 0, 0, 0
	}
	primaryUsed, _ := strconv.ParseFloat(strings.TrimSpace(headers.Get("x-codex-primary-used-percent")), 64)
	secondaryUsed, _ := strconv.ParseFloat(strings.TrimSpace(headers.Get("x-codex-secondary-used-percent")), 64)
	primaryWindow, _ := strconv.Atoi(strings.TrimSpace(headers.Get("x-codex-primary-window-minutes")))
	secondaryWindow, _ := strconv.Atoi(strings.TrimSpace(headers.Get("x-codex-secondary-window-minutes")))
	primaryResetAfter, _ := strconv.Atoi(strings.TrimSpace(headers.Get("x-codex-primary-reset-after-seconds")))
	secondaryResetAfter, _ := strconv.Atoi(strings.TrimSpace(headers.Get("x-codex-secondary-reset-after-seconds")))
	return primaryUsed, secondaryUsed, primaryWindow, secondaryWindow, primaryResetAfter, secondaryResetAfter
}

func maybeWarnOpenAICodexProbeDegenerate(account *Account, updates map[string]any) {
	if account == nil || !isOpenAIProPlan(account) || len(updates) == 0 {
		return
	}
	normal5hRaw, normal5hOK := updates["codex_5h_used_percent"]
	normal7dRaw, normal7dOK := updates["codex_7d_used_percent"]
	spark5hRaw, spark5hOK := updates[codexSpark5hUsedPercentKey]
	spark7dRaw, spark7dOK := updates[codexSpark7dUsedPercentKey]
	if !normal5hOK || !normal7dOK || !spark5hOK || !spark7dOK {
		return
	}

	normal5h := parseExtraFloat64(normal5hRaw)
	normal7d := parseExtraFloat64(normal7dRaw)
	spark5h := parseExtraFloat64(spark5hRaw)
	spark7d := parseExtraFloat64(spark7dRaw)

	normal5hResetAt, okNormal5hReset := parseExtraTimeRFC3339(updates["codex_5h_reset_at"])
	normal7dResetAt, okNormal7dReset := parseExtraTimeRFC3339(updates["codex_7d_reset_at"])
	spark5hResetAt, okSpark5hReset := parseExtraTimeRFC3339(updates[codexSpark5hResetAtKey])
	spark7dResetAt, okSpark7dReset := parseExtraTimeRFC3339(updates[codexSpark7dResetAtKey])
	if !okNormal5hReset || !okNormal7dReset || !okSpark5hReset || !okSpark7dReset {
		return
	}

	floatEq := func(a, b float64) bool { return math.Abs(a-b) < 1e-9 }
	timeEq := func(a, b time.Time) bool { return a.UTC().Equal(b.UTC()) }

	if floatEq(normal5h, spark5h) && floatEq(normal7d, spark7d) && timeEq(normal5hResetAt, spark5hResetAt) && timeEq(normal7dResetAt, spark7dResetAt) {
		slog.Warn(
			"openai_codex_probe_degenerate",
			"account_id", account.ID,
			"codex_5h_used_percent", normal5h,
			"codex_7d_used_percent", normal7d,
			"codex_spark_5h_used_percent", spark5h,
			"codex_spark_7d_used_percent", spark7d,
			"codex_5h_reset_at", normal5hResetAt.UTC().Format(time.RFC3339),
			"codex_7d_reset_at", normal7dResetAt.UTC().Format(time.RFC3339),
			"codex_spark_5h_reset_at", spark5hResetAt.UTC().Format(time.RFC3339),
			"codex_spark_7d_reset_at", spark7dResetAt.UTC().Format(time.RFC3339),
		)
	}
}

func parseExtraTimeRFC3339(raw any) (time.Time, bool) {
	if raw == nil {
		return time.Time{}, false
	}
	ts, err := parseTime(strings.TrimSpace(parseExtraString(raw)))
	if err != nil {
		return time.Time{}, false
	}
	return ts, true
}
