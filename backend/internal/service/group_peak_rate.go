package service

import (
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/timezone"
)

const groupPeakTimeLayout = "15:04"

func NormalizeGroupPeakRateConfig(subscriptionType string, enabled bool, start, end string, multiplier float64) (bool, string, string, float64, error) {
	if subscriptionType != SubscriptionTypeSubscription || !enabled {
		return false, "", "", 1.0, nil
	}
	start = strings.TrimSpace(start)
	end = strings.TrimSpace(end)
	startMinute, err := parseGroupPeakMinute(start)
	if err != nil {
		return false, "", "", 1.0, fmt.Errorf("invalid peak_start: %w", err)
	}
	endMinute, err := parseGroupPeakMinute(end)
	if err != nil {
		return false, "", "", 1.0, fmt.Errorf("invalid peak_end: %w", err)
	}
	if startMinute >= endMinute {
		return false, "", "", 1.0, fmt.Errorf("peak_start must be earlier than peak_end")
	}
	if multiplier < 0 {
		return false, "", "", 1.0, fmt.Errorf("peak_rate_multiplier cannot be negative")
	}
	return true, start, end, multiplier, nil
}

func (g *Group) IsPeakRateActiveAt(now time.Time) bool {
	if g == nil || !g.PeakRateEnabled || !g.IsSubscriptionType() {
		return false
	}
	startMinute, err := parseGroupPeakMinute(g.PeakStart)
	if err != nil {
		return false
	}
	endMinute, err := parseGroupPeakMinute(g.PeakEnd)
	if err != nil || startMinute >= endMinute {
		return false
	}
	local := now.In(timezone.Location())
	currentMinute := local.Hour()*60 + local.Minute()
	return currentMinute >= startMinute && currentMinute < endMinute
}

func (g *Group) EffectiveTokenRateMultiplierAt(base float64, now time.Time) float64 {
	effective := g.EffectiveFlatRateMultiplier(base)
	if g == nil || !g.IsPeakRateActiveAt(now) {
		return effective
	}
	if g.PeakRateMultiplier < 0 {
		return effective
	}
	return effective * g.PeakRateMultiplier
}

func (g *Group) EffectiveFlatRateMultiplier(base float64) float64 {
	if g == nil || !g.ProfitControlEnabled {
		return base
	}
	enabled, minMargin, safetyBuffer := NormalizeGroupProfitConfig(true, g.ProfitMinMargin, g.ProfitSafetyBuffer)
	if !enabled {
		return base
	}
	return base * (1 + (minMargin+safetyBuffer)/100)
}

func ServerPeakRateTimezoneName() string {
	return timezone.Name()
}

func ServerPeakRateUTCOffset(now time.Time) string {
	if now.IsZero() {
		now = time.Now()
	}
	_, offset := now.In(timezone.Location()).Zone()
	sign := "+"
	if offset < 0 {
		sign = "-"
		offset = -offset
	}
	return fmt.Sprintf("%s%02d:%02d", sign, offset/3600, (offset%3600)/60)
}

func parseGroupPeakMinute(value string) (int, error) {
	value = strings.TrimSpace(value)
	parsed, err := time.Parse(groupPeakTimeLayout, value)
	if err != nil {
		return 0, err
	}
	if parsed.Format(groupPeakTimeLayout) != value {
		return 0, fmt.Errorf("must use HH:MM")
	}
	return parsed.Hour()*60 + parsed.Minute(), nil
}
