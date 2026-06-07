package service

import (
	"context"
	"time"
)

const defaultPeriodicJobLeaderLockTTL = 2 * time.Minute
const maxPeriodicJobLeaderLockTTL = 30 * time.Minute

type PeriodicJobLeaderGate interface {
	RunIfLeader(ctx context.Context, jobName string, ttl time.Duration, run func(context.Context)) bool
}

func periodicJobLeaderTTL(interval time.Duration) time.Duration {
	ttl := interval * 5
	if ttl < defaultPeriodicJobLeaderLockTTL {
		return defaultPeriodicJobLeaderLockTTL
	}
	if ttl > maxPeriodicJobLeaderLockTTL {
		return maxPeriodicJobLeaderLockTTL
	}
	return ttl
}
