package service

import (
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	PublicModelCatalogDiscountWindowOnce  = "once"
	PublicModelCatalogDiscountWindowDaily = "daily"

	defaultPublicModelCatalogDiscountTimezone = DefaultTimeAccessPolicyTimezone
	publicModelCatalogDiscountPrecision       = 1e8
)

type publicModelCatalogDiscountEvaluation struct {
	Status  PublicModelCatalogDiscountStatus
	Policy  *PublicModelCatalogDiscountPolicy
	Matched *PublicModelCatalogDiscountWindow
}

func normalizePublicModelCatalogDiscountPolicy(
	policy *PublicModelCatalogDiscountPolicy,
) (*PublicModelCatalogDiscountPolicy, error) {
	if policy == nil || publicModelCatalogDiscountPolicyIsZero(policy) {
		return nil, nil
	}
	out := &PublicModelCatalogDiscountPolicy{
		Enabled:          policy.Enabled,
		ReductionPercent: policy.ReductionPercent,
		Timezone:         strings.TrimSpace(policy.Timezone),
	}
	if out.Timezone == "" {
		out.Timezone = defaultPublicModelCatalogDiscountTimezone
	}
	if _, err := time.LoadLocation(out.Timezone); err != nil {
		return nil, fmt.Errorf("invalid discount timezone %q: %w", out.Timezone, err)
	}
	if out.Enabled {
		if out.ReductionPercent <= 0 || out.ReductionPercent > 100 {
			return nil, fmt.Errorf("discount reduction_percent must be > 0 and <= 100")
		}
	}
	windows, err := normalizePublicModelCatalogDiscountWindows(policy.Windows)
	if err != nil {
		return nil, err
	}
	out.Windows = windows
	if !out.Enabled {
		out.Windows = nil
		out.ReductionPercent = 0
	}
	return out, nil
}

func publicModelCatalogDiscountPolicyInputError(err error) error {
	if err == nil {
		return nil
	}
	return infraerrors.BadRequest(
		"PUBLIC_MODEL_DISCOUNT_POLICY_INVALID",
		"限时折扣配置无效，请检查减免比例、时区和时间窗口",
	).WithCause(err)
}

func publicModelCatalogDiscountPolicyIsZero(policy *PublicModelCatalogDiscountPolicy) bool {
	return policy == nil || (!policy.Enabled && policy.ReductionPercent == 0 &&
		strings.TrimSpace(policy.Timezone) == "" && len(policy.Windows) == 0)
}

func normalizePublicModelCatalogDiscountWindows(
	windows []PublicModelCatalogDiscountWindow,
) ([]PublicModelCatalogDiscountWindow, error) {
	if len(windows) == 0 {
		return nil, nil
	}
	out := make([]PublicModelCatalogDiscountWindow, 0, len(windows))
	for _, window := range windows {
		normalized, err := normalizePublicModelCatalogDiscountWindow(window)
		if err != nil {
			return nil, err
		}
		out = append(out, normalized)
	}
	return out, nil
}

func normalizePublicModelCatalogDiscountWindow(
	window PublicModelCatalogDiscountWindow,
) (PublicModelCatalogDiscountWindow, error) {
	kind := strings.TrimSpace(strings.ToLower(window.Type))
	normalized := PublicModelCatalogDiscountWindow{
		ID:   strings.TrimSpace(window.ID),
		Type: kind,
	}
	switch kind {
	case PublicModelCatalogDiscountWindowOnce:
		start, end, err := parsePublicModelCatalogOnceDiscountWindow(window)
		if err != nil {
			return PublicModelCatalogDiscountWindow{}, err
		}
		normalized.StartAt = start.Format(time.RFC3339)
		normalized.EndAt = end.Format(time.RFC3339)
	case PublicModelCatalogDiscountWindowDaily:
		start, end, err := parsePublicModelCatalogDailyDiscountWindow(window)
		if err != nil {
			return PublicModelCatalogDiscountWindow{}, err
		}
		normalized.StartTime = formatPublicModelCatalogDiscountClock(start)
		normalized.EndTime = formatPublicModelCatalogDiscountClock(end)
		normalized.Days = normalizePublicModelCatalogDiscountDays(window.Days)
		if len(normalized.Days) == 0 {
			return PublicModelCatalogDiscountWindow{}, fmt.Errorf("discount daily window days cannot be empty")
		}
	default:
		return PublicModelCatalogDiscountWindow{}, fmt.Errorf("discount window type must be once or daily")
	}
	return normalized, nil
}

func parsePublicModelCatalogOnceDiscountWindow(
	window PublicModelCatalogDiscountWindow,
) (time.Time, time.Time, error) {
	start, err := time.Parse(time.RFC3339, strings.TrimSpace(window.StartAt))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid discount start_at %q: %w", window.StartAt, err)
	}
	end, err := time.Parse(time.RFC3339, strings.TrimSpace(window.EndAt))
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid discount end_at %q: %w", window.EndAt, err)
	}
	if !end.After(start) {
		return time.Time{}, time.Time{}, fmt.Errorf("discount end_at must be after start_at")
	}
	return start.UTC(), end.UTC(), nil
}

func parsePublicModelCatalogDailyDiscountWindow(
	window PublicModelCatalogDiscountWindow,
) (int, int, error) {
	start, err := parsePublicModelCatalogDiscountClock(window.StartTime)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid discount start_time %q: %w", window.StartTime, err)
	}
	end, err := parsePublicModelCatalogDiscountClock(window.EndTime)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid discount end_time %q: %w", window.EndTime, err)
	}
	if start == end {
		return 0, 0, fmt.Errorf("discount daily window start_time and end_time cannot be equal")
	}
	return start, end, nil
}

func parsePublicModelCatalogDiscountClock(value string) (int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 3 {
		return 0, fmt.Errorf("expected HH:mm:ss")
	}
	parsed, err := time.Parse("15:04:05", strings.Join(parts, ":"))
	if err != nil {
		return 0, err
	}
	return parsed.Hour()*3600 + parsed.Minute()*60 + parsed.Second(), nil
}

func formatPublicModelCatalogDiscountClock(seconds int) string {
	seconds = ((seconds % (24 * 3600)) + (24 * 3600)) % (24 * 3600)
	return fmt.Sprintf("%02d:%02d:%02d", seconds/3600, (seconds%3600)/60, seconds%60)
}

func normalizePublicModelCatalogDiscountDays(days []int) []int {
	seen := map[int]struct{}{}
	for _, day := range days {
		if day < 0 || day > 6 {
			continue
		}
		seen[day] = struct{}{}
	}
	out := make([]int, 0, len(seen))
	for day := range seen {
		out = append(out, day)
	}
	sort.Ints(out)
	return out
}

func evaluatePublicModelCatalogDiscount(
	policy *PublicModelCatalogDiscountPolicy,
	completedAt time.Time,
) publicModelCatalogDiscountEvaluation {
	normalized, err := normalizePublicModelCatalogDiscountPolicy(policy)
	if err != nil || normalized == nil || !normalized.Enabled {
		return publicModelCatalogDiscountEvaluation{}
	}
	completedAt = ceilPublicModelCatalogDiscountCompletedAt(completedAt)
	status := PublicModelCatalogDiscountStatus{
		Active:           false,
		ReductionPercent: normalized.ReductionPercent,
		Timezone:         normalized.Timezone,
		CompletedAt:      completedAt.UTC().Format(time.RFC3339),
	}
	for _, window := range normalized.Windows {
		if publicModelCatalogDiscountWindowContains(normalized.Timezone, window, completedAt) {
			status.Active = true
			status.WindowID = strings.TrimSpace(window.ID)
			status.WindowType = strings.TrimSpace(window.Type)
			return publicModelCatalogDiscountEvaluation{Status: status, Policy: normalized, Matched: &window}
		}
	}
	return publicModelCatalogDiscountEvaluation{Status: status, Policy: normalized}
}

func ceilPublicModelCatalogDiscountCompletedAt(t time.Time) time.Time {
	if t.IsZero() {
		t = time.Now()
	}
	t = t.UTC()
	if t.Nanosecond() == 0 {
		return t
	}
	return t.Truncate(time.Second).Add(time.Second)
}

func publicModelCatalogDiscountWindowContains(
	timezone string,
	window PublicModelCatalogDiscountWindow,
	completedAt time.Time,
) bool {
	switch strings.TrimSpace(window.Type) {
	case PublicModelCatalogDiscountWindowOnce:
		start, end, err := parsePublicModelCatalogOnceDiscountWindow(window)
		if err != nil {
			return false
		}
		return !completedAt.Before(start) && completedAt.Before(end)
	case PublicModelCatalogDiscountWindowDaily:
		start, end, err := parsePublicModelCatalogDailyDiscountWindow(window)
		if err != nil {
			return false
		}
		loc, err := time.LoadLocation(timezone)
		if err != nil {
			return false
		}
		local := completedAt.In(loc)
		return publicModelCatalogDiscountDailyWindowContains(window.Days, start, end, local)
	default:
		return false
	}
}

func publicModelCatalogDiscountDailyWindowContains(days []int, start, end int, t time.Time) bool {
	nowSecond := t.Hour()*3600 + t.Minute()*60 + t.Second()
	weekday := int(t.Weekday())
	prevWeekday := (weekday + 6) % 7
	for _, day := range normalizePublicModelCatalogDiscountDays(days) {
		if start < end {
			if day == weekday && nowSecond >= start && nowSecond < end {
				return true
			}
			continue
		}
		if day == weekday && nowSecond >= start {
			return true
		}
		if day == prevWeekday && nowSecond < end {
			return true
		}
	}
	return false
}

func applyPublicModelCatalogDiscountToPriceDisplay(
	display PublicModelCatalogPriceDisplay,
	status PublicModelCatalogDiscountStatus,
) PublicModelCatalogPriceDisplay {
	display = normalizePublicModelCatalogPriceDisplay(display)
	if !status.Active || status.ReductionPercent <= 0 {
		return display
	}
	return PublicModelCatalogPriceDisplay{
		Primary:   applyPublicModelCatalogDiscountToPriceEntries(display.Primary, status.ReductionPercent),
		Secondary: applyPublicModelCatalogDiscountToPriceEntries(display.Secondary, status.ReductionPercent),
	}
}

func applyPublicModelCatalogDiscountToPriceEntries(
	entries []PublicModelCatalogPriceEntry,
	reductionPercent float64,
) []PublicModelCatalogPriceEntry {
	if len(entries) == 0 {
		return nil
	}
	out := make([]PublicModelCatalogPriceEntry, 0, len(entries))
	for _, entry := range entries {
		next := normalizePublicModelCatalogPriceEntryCompat(entry)
		if next.Configured && next.Value > 0 {
			next.Value = ceilPublicModelCatalogDiscountAmount(next.Value * (100 - reductionPercent) / 100)
		}
		out = append(out, next)
	}
	return out
}

func applyPublicModelCatalogDiscountToImageFixedPricing(
	pricing PublicModelImageFixedPricing,
	status PublicModelCatalogDiscountStatus,
) PublicModelImageFixedPricing {
	pricing = normalizePublicModelImageFixedPricing(pricing)
	if !status.Active || status.ReductionPercent <= 0 || len(pricing.Prices) == 0 {
		return pricing
	}
	for key, value := range pricing.Prices {
		if value == nil || *value <= 0 {
			continue
		}
		discounted := ceilPublicModelCatalogDiscountAmount(*value * (100 - status.ReductionPercent) / 100)
		pricing.Prices[key] = &discounted
	}
	return pricing
}

func ceilPublicModelCatalogDiscountAmount(value float64) float64 {
	if value <= 0 || !isFinitePublicModelCatalogDiscountAmount(value) {
		return 0
	}
	return math.Ceil(value*publicModelCatalogDiscountPrecision) / publicModelCatalogDiscountPrecision
}

func isFinitePublicModelCatalogDiscountAmount(value float64) bool {
	return !math.IsNaN(value) && !math.IsInf(value, 0)
}

func clonePublicModelCatalogDiscountPolicy(
	policy *PublicModelCatalogDiscountPolicy,
) *PublicModelCatalogDiscountPolicy {
	if policy == nil {
		return nil
	}
	cloned := *policy
	cloned.Windows = make([]PublicModelCatalogDiscountWindow, 0, len(policy.Windows))
	for _, window := range policy.Windows {
		next := window
		next.Days = append([]int(nil), window.Days...)
		cloned.Windows = append(cloned.Windows, next)
	}
	return &cloned
}

func clonePublicModelCatalogDiscountStatus(
	status *PublicModelCatalogDiscountStatus,
) *PublicModelCatalogDiscountStatus {
	if status == nil {
		return nil
	}
	cloned := *status
	return &cloned
}
