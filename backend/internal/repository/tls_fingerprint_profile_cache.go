package repository

import (
	"context"
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/redis/go-redis/v9"
)

const (
	tlsFingerprintProfileCacheKey      = "tls_fingerprint_profiles"
	tlsFingerprintProfilePubSubChannel = "tls_fingerprint_profiles_updated"
	tlsFingerprintProfileCacheTTL      = 24 * time.Hour
)

type tlsFingerprintProfileCache struct {
	rdb        *redis.Client
	localCache []*model.TLSFingerprintProfile
	localMu    sync.RWMutex
}

func NewTLSFingerprintProfileCache(rdb *redis.Client) service.TLSFingerprintProfileCache {
	return &tlsFingerprintProfileCache{rdb: rdb}
}

func (c *tlsFingerprintProfileCache) Get(ctx context.Context) ([]*model.TLSFingerprintProfile, bool) {
	c.localMu.RLock()
	if c.localCache != nil {
		profiles := c.localCache
		c.localMu.RUnlock()
		return profiles, true
	}
	c.localMu.RUnlock()

	if c.rdb == nil {
		return nil, false
	}
	data, err := c.rdb.Get(ctx, tlsFingerprintProfileCacheKey).Bytes()
	if err != nil {
		if err != redis.Nil {
			slog.Warn("tls_fingerprint_profile_cache_get_failed", "error", err)
		}
		return nil, false
	}

	var profiles []*model.TLSFingerprintProfile
	if err := json.Unmarshal(data, &profiles); err != nil {
		slog.Warn("tls_fingerprint_profile_cache_unmarshal_failed", "error", err)
		return nil, false
	}

	c.localMu.Lock()
	c.localCache = profiles
	c.localMu.Unlock()
	return profiles, true
}

func (c *tlsFingerprintProfileCache) Set(ctx context.Context, profiles []*model.TLSFingerprintProfile) error {
	c.localMu.Lock()
	c.localCache = profiles
	c.localMu.Unlock()

	if c.rdb == nil {
		return nil
	}
	data, err := json.Marshal(profiles)
	if err != nil {
		return err
	}
	return c.rdb.Set(ctx, tlsFingerprintProfileCacheKey, data, tlsFingerprintProfileCacheTTL).Err()
}

func (c *tlsFingerprintProfileCache) Invalidate(ctx context.Context) error {
	c.localMu.Lock()
	c.localCache = nil
	c.localMu.Unlock()

	if c.rdb == nil {
		return nil
	}
	return c.rdb.Del(ctx, tlsFingerprintProfileCacheKey).Err()
}

func (c *tlsFingerprintProfileCache) NotifyUpdate(ctx context.Context) error {
	if c.rdb == nil {
		return nil
	}
	return c.rdb.Publish(ctx, tlsFingerprintProfilePubSubChannel, "refresh").Err()
}

func (c *tlsFingerprintProfileCache) SubscribeUpdates(ctx context.Context, handler func()) {
	if c.rdb == nil {
		return
	}
	go func() {
		sub := c.rdb.Subscribe(ctx, tlsFingerprintProfilePubSubChannel)
		defer func() { _ = sub.Close() }()

		for {
			select {
			case <-ctx.Done():
				return
			case msg := <-sub.Channel():
				if msg == nil {
					return
				}
				c.localMu.Lock()
				c.localCache = nil
				c.localMu.Unlock()
				if handler != nil {
					handler()
				}
			}
		}
	}()
}
