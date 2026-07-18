package service

import (
	"strings"
	"time"
)

const openAIQuotaShortWindowMaxSeconds = 6 * 60 * 60

type openAIQuotaWindowPair struct {
	fiveHour *OpenAIQuotaWindowSnapshot
	sevenDay *OpenAIQuotaWindowSnapshot
}

func openAIQuotaWindowPairFromRateLimit(rateLimit *OpenAIRateLimit, now time.Time) openAIQuotaWindowPair {
	if rateLimit == nil {
		return openAIQuotaWindowPair{}
	}

	primary := openAIQuotaWindowSnapshot(rateLimit.PrimaryWindow, now)
	secondary := openAIQuotaWindowSnapshot(rateLimit.SecondaryWindow, now)
	switch {
	case primary == nil:
		return classifySingleOpenAIQuotaWindow(secondary)
	case secondary == nil:
		return classifySingleOpenAIQuotaWindow(primary)
	case primary.LimitWindowSeconds <= secondary.LimitWindowSeconds:
		return openAIQuotaWindowPair{fiveHour: primary, sevenDay: secondary}
	default:
		return openAIQuotaWindowPair{fiveHour: secondary, sevenDay: primary}
	}
}

func classifySingleOpenAIQuotaWindow(window *OpenAIQuotaWindowSnapshot) openAIQuotaWindowPair {
	if window == nil || window.LimitWindowSeconds <= 0 {
		return openAIQuotaWindowPair{}
	}
	if window.LimitWindowSeconds <= openAIQuotaShortWindowMaxSeconds {
		return openAIQuotaWindowPair{fiveHour: window}
	}
	return openAIQuotaWindowPair{sevenDay: window}
}

func openAIQuotaWindowSnapshot(window *OpenAIRateLimitWindow, now time.Time) *OpenAIQuotaWindowSnapshot {
	if window == nil || window.LimitWindowSeconds <= 0 {
		return nil
	}

	progress := &UsageProgress{
		Utilization:      window.UsedPercent,
		RemainingSeconds: int(maxInt64(window.ResetAfterSeconds, 0)),
	}
	if resetAt := openAIQuotaResetAt(window, now); resetAt != nil {
		progress.ResetsAt = resetAt
		if progress.RemainingSeconds == 0 {
			progress.RemainingSeconds = int(maxInt64(int64(time.Until(*resetAt).Seconds()), 0))
		}
		if !now.Before(*resetAt) {
			progress.Utilization = 0
			progress.RemainingSeconds = 0
		}
	}

	return &OpenAIQuotaWindowSnapshot{
		Progress:           progress,
		LimitWindowSeconds: window.LimitWindowSeconds,
	}
}

func openAIQuotaResetAt(window *OpenAIRateLimitWindow, now time.Time) *time.Time {
	switch {
	case window == nil:
		return nil
	case window.ResetAt > 1_000_000_000_000:
		resetAt := time.UnixMilli(window.ResetAt).UTC()
		return &resetAt
	case window.ResetAt > 0:
		resetAt := time.Unix(window.ResetAt, 0).UTC()
		return &resetAt
	case window.ResetAfterSeconds > 0:
		resetAt := now.UTC().Add(time.Duration(window.ResetAfterSeconds) * time.Second)
		return &resetAt
	default:
		return nil
	}
}

func isOpenAIQuotaSparkLimit(limit OpenAIAdditionalRateLimit) bool {
	name := strings.ToLower(strings.TrimSpace(limit.LimitName))
	feature := strings.ToLower(strings.TrimSpace(limit.MeteredFeature))
	return strings.Contains(name, "spark") || strings.Contains(feature, "spark")
}

func applyOpenAIQuotaWindowsFromSnapshot(usage *UsageInfo, snapshot *OpenAIResetCreditsSnapshot) {
	if usage == nil || snapshot == nil {
		return
	}
	if snapshot.FiveHour != nil && snapshot.FiveHour.Progress != nil {
		usage.FiveHour = snapshot.FiveHour.Progress
	}
	if snapshot.SevenDay != nil && snapshot.SevenDay.Progress != nil {
		usage.SevenDay = snapshot.SevenDay.Progress
	}
	if snapshot.SparkFiveHour != nil && snapshot.SparkFiveHour.Progress != nil {
		usage.SparkFiveHour = snapshot.SparkFiveHour.Progress
	}
	if snapshot.SparkSevenDay != nil && snapshot.SparkSevenDay.Progress != nil {
		usage.SparkSevenDay = snapshot.SparkSevenDay.Progress
	}
	if !snapshot.UpdatedAt.IsZero() {
		updatedAt := snapshot.UpdatedAt.UTC()
		usage.UpdatedAt = &updatedAt
	}
}

func maxInt64(left, right int64) int64 {
	if left > right {
		return left
	}
	return right
}
