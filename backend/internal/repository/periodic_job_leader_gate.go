package repository

import (
	"context"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/config"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const defaultPeriodicJobLeaderLockTTL = 2 * time.Minute

var periodicJobLeaderRenewScript = redis.NewScript(`
if redis.call("GET", KEYS[1]) == ARGV[1] then
  return redis.call("PEXPIRE", KEYS[1], ARGV[2])
end
return 0
`)

type redisPeriodicJobLeaderGate struct {
	redisClient *redis.Client
	instanceID  string
	enabled     bool

	warnNoRedisOnce sync.Once
	warnSetNXOnce   sync.Map
}

func NewPeriodicJobLeaderGate(redisClient *redis.Client, cfg *config.Config) service.PeriodicJobLeaderGate {
	enabled := cfg == nil || strings.TrimSpace(cfg.RunMode) != config.RunModeSimple
	return &redisPeriodicJobLeaderGate{
		redisClient: redisClient,
		instanceID:  uuid.NewString(),
		enabled:     enabled,
	}
}

func (g *redisPeriodicJobLeaderGate) RunIfLeader(ctx context.Context, jobName string, ttl time.Duration, run func(context.Context)) bool {
	if run == nil {
		return false
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if g == nil || !g.enabled {
		run(ctx)
		return true
	}
	if g.redisClient == nil {
		g.warnNoRedisOnce.Do(func() {
			slog.Warn("periodic_job_leader_lock_missing_redis", "job", jobName)
		})
		return false
	}
	if ttl <= 0 {
		ttl = defaultPeriodicJobLeaderLockTTL
	}
	key := periodicJobLeaderLockKey(jobName)
	if !g.tryAcquireOrRenew(ctx, key, jobName, ttl) {
		return false
	}
	run(ctx)
	return true
}

func (g *redisPeriodicJobLeaderGate) tryAcquireOrRenew(ctx context.Context, key, jobName string, ttl time.Duration) bool {
	acquired, err := g.redisClient.SetNX(ctx, key, g.instanceID, ttl).Result()
	if err != nil {
		g.warnSetNXOnceFor(jobName, err)
		return false
	}
	if acquired {
		return true
	}
	renewed, err := periodicJobLeaderRenewScript.Run(
		ctx,
		g.redisClient,
		[]string{key},
		g.instanceID,
		periodicJobLeaderMaxInt64(1, ttl.Milliseconds()),
	).Int()
	if err != nil {
		g.warnSetNXOnceFor(jobName, err)
		return false
	}
	return renewed == 1
}

func (g *redisPeriodicJobLeaderGate) warnSetNXOnceFor(jobName string, err error) {
	normalized := strings.TrimSpace(jobName)
	if normalized == "" {
		normalized = "unknown"
	}
	onceValue, _ := g.warnSetNXOnce.LoadOrStore(normalized, &sync.Once{})
	once, _ := onceValue.(*sync.Once)
	if once == nil {
		slog.Warn("periodic_job_leader_lock_acquire_failed", "job", normalized, "error", err)
		return
	}
	once.Do(func() {
		slog.Warn("periodic_job_leader_lock_acquire_failed", "job", normalized, "error", err)
	})
}

func periodicJobLeaderLockKey(jobName string) string {
	name := strings.TrimSpace(jobName)
	if name == "" {
		name = "unknown"
	}
	name = strings.NewReplacer(" ", "_", ":", "_", "/", "_").Replace(name)
	return "sub2api:periodic-job:leader:" + name
}

func periodicJobLeaderMaxInt64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
