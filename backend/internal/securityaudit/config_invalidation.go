package securityaudit

import (
	"context"
	"errors"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

type ConfigInvalidator interface {
	Publish(ctx context.Context) error
	Subscribe(ctx context.Context, handler func()) error
}

type RedisConfigInvalidator struct {
	client *redis.Client
}

func NewRedisConfigInvalidator(client *redis.Client) *RedisConfigInvalidator {
	return &RedisConfigInvalidator{client: client}
}

func (i *RedisConfigInvalidator) Publish(ctx context.Context) error {
	if i == nil || i.client == nil {
		return nil
	}
	return i.client.Publish(ctx, ConfigInvalidationChannel, "reload").Err()
}

func (i *RedisConfigInvalidator) Subscribe(ctx context.Context, handler func()) error {
	if i == nil || i.client == nil {
		return errors.New("prompt audit config invalidator unavailable")
	}
	sub := i.client.Subscribe(ctx, ConfigInvalidationChannel)
	defer func() {
		if err := sub.Close(); err != nil {
			promptAuditLog(ctx).Warn("prompt_audit_config_subscription_close_failed",
				zap.String("error_code", "prompt_audit_config_subscription_close_failed"),
			)
		}
	}()
	ch := sub.Channel()
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-ch:
			if !ok {
				return nil
			}
			if handler != nil {
				handler()
			}
		}
	}
}
