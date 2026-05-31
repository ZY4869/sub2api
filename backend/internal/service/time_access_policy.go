package service

import (
	"fmt"
	"sort"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const DefaultTimeAccessPolicyTimezone = "Asia/Singapore"

const (
	TimeAccessDecisionAllowed       = "allowed"
	TimeAccessDecisionDisabled      = "disabled"
	TimeAccessDecisionNotBefore     = "not_before"
	TimeAccessDecisionNotAfter      = "not_after"
	TimeAccessDecisionOutsideWindow = "outside_window"
	TimeAccessDecisionInvalidPolicy = "invalid_policy"
	TimeAccessDecisionDailyLimit    = "daily_allowed_minutes"
)

type TimeAccessPolicy struct {
	Enabled             bool               `json:"enabled"`
	Timezone            string             `json:"timezone,omitempty"`
	NotBefore           *time.Time         `json:"not_before,omitempty"`
	NotAfter            *time.Time         `json:"not_after,omitempty"`
	WeeklyWindows       []TimeAccessWindow `json:"weekly_windows,omitempty"`
	DailyAllowedMinutes *int               `json:"daily_allowed_minutes,omitempty"`
}

type TimeAccessWindow struct {
	Days  []int  `json:"days"`
	Start string `json:"start"`
	End   string `json:"end"`
}

type TimeAccessEvaluation struct {
	Allowed bool
	Reason  string
	Policy  *TimeAccessPolicy
}

func (p *TimeAccessPolicy) IsZero() bool {
	return p == nil || (!p.Enabled && strings.TrimSpace(p.Timezone) == "" &&
		p.NotBefore == nil && p.NotAfter == nil && len(p.WeeklyWindows) == 0 &&
		p.DailyAllowedMinutes == nil)
}

func NormalizeTimeAccessPolicy(policy *TimeAccessPolicy) (*TimeAccessPolicy, error) {
	if policy == nil || policy.IsZero() {
		return nil, nil
	}
	out := *policy
	if strings.TrimSpace(out.Timezone) == "" {
		out.Timezone = DefaultTimeAccessPolicyTimezone
	} else {
		out.Timezone = strings.TrimSpace(out.Timezone)
	}
	if _, err := time.LoadLocation(out.Timezone); err != nil {
		return nil, fmt.Errorf("invalid time access timezone %q: %w", out.Timezone, err)
	}
	if out.NotBefore != nil && out.NotAfter != nil && out.NotAfter.Before(*out.NotBefore) {
		return nil, fmt.Errorf("time access not_after must be after not_before")
	}
	if out.DailyAllowedMinutes != nil {
		if *out.DailyAllowedMinutes < 0 || *out.DailyAllowedMinutes > 24*60 {
			return nil, fmt.Errorf("daily_allowed_minutes must be between 0 and 1440")
		}
	}
	windows, err := normalizeTimeAccessWindows(out.WeeklyWindows)
	if err != nil {
		return nil, err
	}
	out.WeeklyWindows = windows
	if out.DailyAllowedMinutes != nil && timeAccessDailyWindowMinutes(windows) > *out.DailyAllowedMinutes {
		return nil, fmt.Errorf("weekly_windows exceed daily_allowed_minutes")
	}
	return &out, nil
}

func timeAccessPolicyInputError(err error) error {
	if err == nil {
		return nil
	}
	return infraerrors.BadRequest(
		"TIME_ACCESS_POLICY_INVALID",
		"时间访问策略无效，请检查时区、时间窗口和每日上限",
	).WithCause(err)
}

func EvaluateTimeAccessPolicy(policy *TimeAccessPolicy, now time.Time) TimeAccessEvaluation {
	normalized, err := NormalizeTimeAccessPolicy(policy)
	if err != nil {
		return TimeAccessEvaluation{Allowed: false, Reason: TimeAccessDecisionInvalidPolicy, Policy: policy}
	}
	if normalized == nil {
		return TimeAccessEvaluation{Allowed: true, Reason: TimeAccessDecisionDisabled, Policy: normalized}
	}
	if !normalized.Enabled {
		return TimeAccessEvaluation{Allowed: true, Reason: TimeAccessDecisionDisabled, Policy: normalized}
	}
	if normalized.NotBefore != nil && now.Before(*normalized.NotBefore) {
		return TimeAccessEvaluation{Allowed: false, Reason: TimeAccessDecisionNotBefore, Policy: normalized}
	}
	if normalized.NotAfter != nil && !now.Before(*normalized.NotAfter) {
		return TimeAccessEvaluation{Allowed: false, Reason: TimeAccessDecisionNotAfter, Policy: normalized}
	}
	if len(normalized.WeeklyWindows) == 0 {
		return TimeAccessEvaluation{Allowed: true, Reason: TimeAccessDecisionAllowed, Policy: normalized}
	}
	loc, _ := time.LoadLocation(normalized.Timezone)
	localNow := now.In(loc)
	if timeAccessWindowContains(normalized.WeeklyWindows, localNow) {
		return TimeAccessEvaluation{Allowed: true, Reason: TimeAccessDecisionAllowed, Policy: normalized}
	}
	return TimeAccessEvaluation{Allowed: false, Reason: TimeAccessDecisionOutsideWindow, Policy: normalized}
}

func EvaluateEffectiveTimeAccess(now time.Time, policies ...*TimeAccessPolicy) TimeAccessEvaluation {
	var effective *TimeAccessPolicy
	for _, policy := range policies {
		if policy == nil || policy.IsZero() || !policy.Enabled {
			continue
		}
		eval := EvaluateTimeAccessPolicy(policy, now)
		if !eval.Allowed {
			return eval
		}
		effective = eval.Policy
	}
	return TimeAccessEvaluation{Allowed: true, Reason: TimeAccessDecisionAllowed, Policy: effective}
}

func ValidateTimeAccessSubset(child, parent *TimeAccessPolicy) error {
	normalizedChild, err := NormalizeTimeAccessPolicy(child)
	if err != nil {
		return err
	}
	normalizedParent, err := NormalizeTimeAccessPolicy(parent)
	if err != nil {
		return err
	}
	if normalizedChild == nil || !normalizedChild.Enabled || normalizedParent == nil || !normalizedParent.Enabled {
		return nil
	}
	if normalizedParent.NotBefore != nil && (normalizedChild.NotBefore == nil || normalizedChild.NotBefore.Before(*normalizedParent.NotBefore)) {
		return fmt.Errorf("time access not_before must not be earlier than user hard limit")
	}
	if normalizedParent.NotAfter != nil && (normalizedChild.NotAfter == nil || normalizedChild.NotAfter.After(*normalizedParent.NotAfter)) {
		return fmt.Errorf("time access not_after must not be later than user hard limit")
	}
	if len(normalizedParent.WeeklyWindows) > 0 && len(normalizedChild.WeeklyWindows) == 0 {
		return fmt.Errorf("time access windows must be within user hard limit")
	}
	for _, childWindow := range normalizedChild.WeeklyWindows {
		if !timeAccessWindowCovered(childWindow, normalizedParent.WeeklyWindows) {
			return fmt.Errorf("time access window %v %s-%s exceeds user hard limit", childWindow.Days, childWindow.Start, childWindow.End)
		}
	}
	if normalizedParent.DailyAllowedMinutes != nil {
		childMinutes := timeAccessDailyWindowMinutes(normalizedChild.WeeklyWindows)
		if childMinutes > *normalizedParent.DailyAllowedMinutes {
			return fmt.Errorf("time access windows exceed user hard daily minutes")
		}
	}
	return nil
}

func normalizeTimeAccessWindows(windows []TimeAccessWindow) ([]TimeAccessWindow, error) {
	if len(windows) == 0 {
		return nil, nil
	}
	out := make([]TimeAccessWindow, 0, len(windows))
	for _, window := range windows {
		start, err := parseTimeAccessClock(window.Start)
		if err != nil {
			return nil, fmt.Errorf("invalid window start %q: %w", window.Start, err)
		}
		end, err := parseTimeAccessClock(window.End)
		if err != nil {
			return nil, fmt.Errorf("invalid window end %q: %w", window.End, err)
		}
		if start == end {
			return nil, fmt.Errorf("time access window start and end cannot be equal")
		}
		days := normalizeTimeAccessDays(window.Days)
		if len(days) == 0 {
			return nil, fmt.Errorf("time access window days cannot be empty")
		}
		out = append(out, TimeAccessWindow{
			Days:  days,
			Start: formatTimeAccessClock(start),
			End:   formatTimeAccessClock(end),
		})
	}
	return out, nil
}

func parseTimeAccessClock(value string) (int, error) {
	parts := strings.Split(strings.TrimSpace(value), ":")
	if len(parts) != 2 {
		return 0, fmt.Errorf("expected HH:mm")
	}
	t, err := time.Parse("15:04", strings.TrimSpace(parts[0])+":"+strings.TrimSpace(parts[1]))
	if err != nil {
		return 0, err
	}
	return t.Hour()*60 + t.Minute(), nil
}

func formatTimeAccessClock(minutes int) string {
	minutes = ((minutes % (24 * 60)) + (24 * 60)) % (24 * 60)
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}

func normalizeTimeAccessDays(days []int) []int {
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

func timeAccessWindowContains(windows []TimeAccessWindow, t time.Time) bool {
	nowMinute := t.Hour()*60 + t.Minute()
	weekday := int(t.Weekday())
	prevWeekday := (weekday + 6) % 7
	for _, window := range windows {
		start, _ := parseTimeAccessClock(window.Start)
		end, _ := parseTimeAccessClock(window.End)
		for _, day := range window.Days {
			if start < end {
				if day == weekday && nowMinute >= start && nowMinute < end {
					return true
				}
				continue
			}
			if day == weekday && nowMinute >= start {
				return true
			}
			if day == prevWeekday && nowMinute < end {
				return true
			}
		}
	}
	return false
}

func timeAccessWindowCovered(child TimeAccessWindow, parents []TimeAccessWindow) bool {
	for minute := 0; minute < 7*24*60; minute++ {
		if !timeAccessWeeklyMinuteInWindow(minute, child) {
			continue
		}
		covered := false
		for _, parent := range parents {
			if timeAccessWeeklyMinuteInWindow(minute, parent) {
				covered = true
				break
			}
		}
		if !covered {
			return false
		}
	}
	return true
}

func timeAccessWeeklyMinuteInWindow(weeklyMinute int, window TimeAccessWindow) bool {
	start, _ := parseTimeAccessClock(window.Start)
	end, _ := parseTimeAccessClock(window.End)
	for _, day := range window.Days {
		base := day * 24 * 60
		if start < end {
			if weeklyMinute >= base+start && weeklyMinute < base+end {
				return true
			}
			continue
		}
		if weeklyMinute >= base+start && weeklyMinute < base+24*60 {
			return true
		}
		nextBase := ((day + 1) % 7) * 24 * 60
		if weeklyMinute >= nextBase && weeklyMinute < nextBase+end {
			return true
		}
	}
	return false
}

func timeAccessDailyWindowMinutes(windows []TimeAccessWindow) int {
	maxMinutes := 0
	for day := 0; day < 7; day++ {
		minutes := 0
		for minute := 0; minute < 24*60; minute++ {
			weeklyMinute := day*24*60 + minute
			for _, window := range windows {
				if timeAccessWeeklyMinuteInWindow(weeklyMinute, window) {
					minutes++
					break
				}
			}
		}
		if minutes > maxMinutes {
			maxMinutes = minutes
		}
	}
	return maxMinutes
}
