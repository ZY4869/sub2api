package service

import (
	"context"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type cachedVersionBounds struct {
	min       string
	max       string
	expiresAt int64
}

var versionBoundsCache atomic.Value
var versionBoundsSF singleflight.Group

const versionBoundsCacheTTL = 60 * time.Second
const versionBoundsErrorTTL = 5 * time.Second
const versionBoundsDBTimeout = 5 * time.Second

func (s *SettingService) GetClaudeCodeVersionBounds(ctx context.Context) (min, max string) {
	if cached, ok := versionBoundsCache.Load().(*cachedVersionBounds); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.min, cached.max
		}
	}

	result, _, _ := versionBoundsSF.Do("version_bounds", func() (any, error) {
		if cached, ok := versionBoundsCache.Load().(*cachedVersionBounds); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached, nil
			}
		}

		if s == nil || s.settingRepo == nil {
			cached := &cachedVersionBounds{
				expiresAt: time.Now().Add(versionBoundsErrorTTL).UnixNano(),
			}
			versionBoundsCache.Store(cached)
			return cached, nil
		}

		dbCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), versionBoundsDBTimeout)
		defer cancel()

		values, err := s.settingRepo.GetMultiple(dbCtx, []string{
			SettingKeyMinClaudeCodeVersion,
			SettingKeyMaxClaudeCodeVersion,
		})
		if err != nil {
			logClaudeCodeVersionBoundsFallback(err)
			cached := &cachedVersionBounds{
				expiresAt: time.Now().Add(versionBoundsErrorTTL).UnixNano(),
			}
			versionBoundsCache.Store(cached)
			return cached, nil
		}

		cached := &cachedVersionBounds{
			min:       values[SettingKeyMinClaudeCodeVersion],
			max:       values[SettingKeyMaxClaudeCodeVersion],
			expiresAt: time.Now().Add(versionBoundsCacheTTL).UnixNano(),
		}
		versionBoundsCache.Store(cached)
		return cached, nil
	})

	bounds, ok := result.(*cachedVersionBounds)
	if !ok || bounds == nil {
		return "", ""
	}
	return bounds.min, bounds.max
}

func (s *SettingService) GetMinClaudeCodeVersion(ctx context.Context) string {
	min, _ := s.GetClaudeCodeVersionBounds(ctx)
	return min
}
