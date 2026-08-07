package service

import (
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
)

func defaultGatewayRateMultiplier(cfg *config.Config) float64 {
	if cfg == nil || cfg.Default.RateMultiplier <= 0 {
		return 1
	}
	return cfg.Default.RateMultiplier
}

func effectiveTokenRateMultiplierAt(base float64, group *Group, now time.Time) float64 {
	if group == nil {
		return base
	}
	return group.EffectiveTokenRateMultiplierAt(base, now)
}

func effectiveFlatRateMultiplier(base float64, group *Group) float64 {
	if group == nil {
		return base
	}
	return group.EffectiveFlatRateMultiplier(base)
}
