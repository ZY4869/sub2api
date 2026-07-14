package servertiming

import (
	"context"

	"github.com/redis/go-redis/v9"
)

type RedisHook struct{}

func (RedisHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (RedisHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		done := Observe(ctx, "redis")
		err := next(ctx, cmd)
		done()
		return err
	}
}

func (RedisHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		done := Observe(ctx, "redis")
		err := next(ctx, cmds)
		done()
		return err
	}
}
