package service

import "time"

// APIKeyRateLimitData holds rate limit usage and window state for an API key.
type APIKeyRateLimitData struct {
	Usage5h           float64
	Usage1d           float64
	Usage7d           float64
	Usage5hByCurrency map[string]float64
	Usage1dByCurrency map[string]float64
	Usage7dByCurrency map[string]float64
	Window5hStart     *time.Time
	Window1dStart     *time.Time
	Window7dStart     *time.Time
}

func (d *APIKeyRateLimitData) EffectiveUsage5h() float64 {
	if IsWindowExpired(d.Window5hStart, RateLimitWindow5h) {
		return 0
	}
	return d.Usage5h
}

func (d *APIKeyRateLimitData) EffectiveUsage5hByCurrency() map[string]float64 {
	if d == nil || IsWindowExpired(d.Window5hStart, RateLimitWindow5h) {
		return nil
	}
	return cloneBillingStringMapFloat64(d.Usage5hByCurrency)
}

func (d *APIKeyRateLimitData) EffectiveUsage1d() float64 {
	if IsWindowExpired(d.Window1dStart, RateLimitWindow1d) {
		return 0
	}
	return d.Usage1d
}

func (d *APIKeyRateLimitData) EffectiveUsage1dByCurrency() map[string]float64 {
	if d == nil || IsWindowExpired(d.Window1dStart, RateLimitWindow1d) {
		return nil
	}
	return cloneBillingStringMapFloat64(d.Usage1dByCurrency)
}

func (d *APIKeyRateLimitData) EffectiveUsage7d() float64 {
	if IsWindowExpired(d.Window7dStart, RateLimitWindow7d) {
		return 0
	}
	return d.Usage7d
}

func (d *APIKeyRateLimitData) EffectiveUsage7dByCurrency() map[string]float64 {
	if d == nil || IsWindowExpired(d.Window7dStart, RateLimitWindow7d) {
		return nil
	}
	return cloneBillingStringMapFloat64(d.Usage7dByCurrency)
}

// APIKeyQuotaUsageState captures the latest quota fields after an atomic quota update.
type APIKeyQuotaUsageState struct {
	QuotaUsed float64
	Quota     float64
	Key       string
	Status    string
}
