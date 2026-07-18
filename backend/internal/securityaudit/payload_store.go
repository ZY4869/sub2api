package securityaudit

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const payloadKeyPrefix = "prompt_audit:payload:"

type PayloadStore interface {
	Set(ctx context.Context, jobID int64, payload PromptPayload, ttl time.Duration) error
	Get(ctx context.Context, jobID int64) (PromptPayload, error)
	Delete(ctx context.Context, jobID int64) error
	Ping(ctx context.Context) error
}

type RedisPayloadStore struct {
	client *redis.Client
}

func NewRedisPayloadStore(client *redis.Client) *RedisPayloadStore {
	return &RedisPayloadStore{client: client}
}

func (s *RedisPayloadStore) Set(ctx context.Context, jobID int64, payload PromptPayload, ttl time.Duration) error {
	if s == nil || s.client == nil || jobID <= 0 {
		return errors.New("prompt audit payload store unavailable")
	}
	if ttl <= 0 {
		ttl = time.Duration(DefaultPayloadTTLSeconds) * time.Second
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, payloadKey(jobID), raw, ttl).Err()
}

func (s *RedisPayloadStore) Get(ctx context.Context, jobID int64) (PromptPayload, error) {
	if s == nil || s.client == nil || jobID <= 0 {
		return PromptPayload{}, errors.New("prompt audit payload store unavailable")
	}
	raw, err := s.client.Get(ctx, payloadKey(jobID)).Bytes()
	if err != nil {
		return PromptPayload{}, err
	}
	var payload PromptPayload
	if err := json.Unmarshal(raw, &payload); err != nil {
		return PromptPayload{}, err
	}
	return payload, nil
}

func (s *RedisPayloadStore) Delete(ctx context.Context, jobID int64) error {
	if s == nil || s.client == nil || jobID <= 0 {
		return nil
	}
	return s.client.Del(ctx, payloadKey(jobID)).Err()
}

func (s *RedisPayloadStore) Ping(ctx context.Context) error {
	if s == nil || s.client == nil {
		return errors.New("prompt audit payload store unavailable")
	}
	return s.client.Ping(ctx).Err()
}

func payloadKey(jobID int64) string {
	return fmt.Sprintf("%s%s", payloadKeyPrefix, strconv.FormatInt(jobID, 10))
}
