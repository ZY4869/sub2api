package service

import (
	"math"
	"time"
)

func (p *openAIWSConnPool) maxConnsHardCap() int {
	if p != nil && p.cfg != nil && p.cfg.Gateway.OpenAIWS.MaxConnsPerAccount > 0 {
		return p.cfg.Gateway.OpenAIWS.MaxConnsPerAccount
	}
	return 8
}

func (p *openAIWSConnPool) dynamicMaxConnsEnabled() bool {
	if p != nil && p.cfg != nil {
		return p.cfg.Gateway.OpenAIWS.DynamicMaxConnsByAccountConcurrencyEnabled
	}
	return false
}

func (p *openAIWSConnPool) modeRouterV2Enabled() bool {
	if p != nil && p.cfg != nil {
		return p.cfg.Gateway.OpenAIWS.ModeRouterV2Enabled
	}
	return false
}

func (p *openAIWSConnPool) maxConnsFactorByAccount(account *Account) float64 {
	if p == nil || p.cfg == nil || account == nil {
		return 1.0
	}
	switch account.Type {
	case AccountTypeOAuth:
		if p.cfg.Gateway.OpenAIWS.OAuthMaxConnsFactor > 0 {
			return p.cfg.Gateway.OpenAIWS.OAuthMaxConnsFactor
		}
	case AccountTypeAPIKey:
		if p.cfg.Gateway.OpenAIWS.APIKeyMaxConnsFactor > 0 {
			return p.cfg.Gateway.OpenAIWS.APIKeyMaxConnsFactor
		}
	}
	return 1.0
}

func (p *openAIWSConnPool) effectiveMaxConnsByAccount(account *Account) int {
	hardCap := p.maxConnsHardCap()
	if hardCap <= 0 {
		return 0
	}
	if p.modeRouterV2Enabled() {
		if account == nil {
			return hardCap
		}
		if account.Concurrency <= 0 {
			return 0
		}
		return account.Concurrency
	}
	if account == nil || !p.dynamicMaxConnsEnabled() {
		return hardCap
	}
	if account.Concurrency <= 0 {
		// 0/-1 等“无限制”并发场景下，仍由全局硬上限兜底。
		return hardCap
	}
	factor := p.maxConnsFactorByAccount(account)
	if factor <= 0 {
		factor = 1.0
	}
	effective := int(math.Ceil(float64(account.Concurrency) * factor))
	if effective < 1 {
		effective = 1
	}
	if effective > hardCap {
		effective = hardCap
	}
	return effective
}

func (p *openAIWSConnPool) minIdlePerAccount() int {
	if p != nil && p.cfg != nil && p.cfg.Gateway.OpenAIWS.MinIdlePerAccount >= 0 {
		return p.cfg.Gateway.OpenAIWS.MinIdlePerAccount
	}
	return 0
}

func (p *openAIWSConnPool) maxIdlePerAccount() int {
	if p != nil && p.cfg != nil && p.cfg.Gateway.OpenAIWS.MaxIdlePerAccount >= 0 {
		return p.cfg.Gateway.OpenAIWS.MaxIdlePerAccount
	}
	return 4
}

func (p *openAIWSConnPool) maxConnAge() time.Duration {
	return openAIWSConnMaxAge
}

func (p *openAIWSConnPool) queueLimitPerConn() int {
	if p != nil && p.cfg != nil && p.cfg.Gateway.OpenAIWS.QueueLimitPerConn > 0 {
		return p.cfg.Gateway.OpenAIWS.QueueLimitPerConn
	}
	return 256
}

func (p *openAIWSConnPool) targetUtilization() float64 {
	if p != nil && p.cfg != nil {
		ratio := p.cfg.Gateway.OpenAIWS.PoolTargetUtilization
		if ratio > 0 && ratio <= 1 {
			return ratio
		}
	}
	return 0.7
}

func (p *openAIWSConnPool) prewarmCooldown() time.Duration {
	if p != nil && p.cfg != nil && p.cfg.Gateway.OpenAIWS.PrewarmCooldownMS > 0 {
		return time.Duration(p.cfg.Gateway.OpenAIWS.PrewarmCooldownMS) * time.Millisecond
	}
	return 0
}

func (p *openAIWSConnPool) shouldSuppressPrewarmLocked(ap *openAIWSAccountPool, now time.Time) bool {
	if ap == nil {
		return true
	}
	if ap.prewarmFails <= 0 {
		return false
	}
	if ap.prewarmFailAt.IsZero() {
		ap.prewarmFails = 0
		return false
	}
	if now.Sub(ap.prewarmFailAt) > openAIWSPrewarmFailureWindow {
		ap.prewarmFails = 0
		ap.prewarmFailAt = time.Time{}
		return false
	}
	return ap.prewarmFails >= openAIWSPrewarmFailureSuppress
}

func (p *openAIWSConnPool) dialTimeout() time.Duration {
	if p != nil && p.cfg != nil && p.cfg.Gateway.OpenAIWS.DialTimeoutSeconds > 0 {
		return time.Duration(p.cfg.Gateway.OpenAIWS.DialTimeoutSeconds) * time.Second
	}
	return 10 * time.Second
}

func (p *openAIWSConnPool) targetConnCountLocked(ap *openAIWSAccountPool, maxConns int) int {
	if ap == nil {
		return 0
	}

	if maxConns <= 0 {
		return 0
	}

	minIdle := p.minIdlePerAccount()
	if minIdle < 0 {
		minIdle = 0
	}
	if minIdle > maxConns {
		minIdle = maxConns
	}

	inflight, waiters := accountPoolLoadLocked(ap)
	utilization := p.targetUtilization()
	demand := inflight + waiters
	if demand <= 0 {
		return minIdle
	}

	target := 1
	if demand > 1 {
		target = int(math.Ceil(float64(demand) / utilization))
	}
	if waiters > 0 && target < len(ap.conns)+1 {
		target = len(ap.conns) + 1
	}
	if target < minIdle {
		target = minIdle
	}
	if target > maxConns {
		target = maxConns
	}
	return target
}
