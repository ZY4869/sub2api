package service

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type cachedBackendMode struct {
	value     bool
	expiresAt int64
}

var backendModeCache atomic.Value
var backendModeSF singleflight.Group

const backendModeCacheTTL = 30 * time.Second

func (s *SettingService) IsBackendModeEnabled(ctx context.Context) bool {
	if cached, ok := backendModeCache.Load().(*cachedBackendMode); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.value
		}
	}

	result, err, _ := backendModeSF.Do("backend_mode_enabled", func() (any, error) {
		if cached, ok := backendModeCache.Load().(*cachedBackendMode); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.value, nil
			}
		}
		if s == nil || s.settingRepo == nil {
			backendModeCache.Store(&cachedBackendMode{value: false, expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano()})
			return false, nil
		}
		value, readErr := s.settingRepo.GetValue(ctx, SettingKeyBackendModeEnabled)
		if readErr != nil {
			if errors.Is(readErr, ErrSettingNotFound) {
				backendModeCache.Store(&cachedBackendMode{value: false, expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano()})
				return false, nil
			}
			backendModeCache.Store(&cachedBackendMode{value: false, expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano()})
			return false, nil
		}
		enabled := strings.TrimSpace(value) == "true"
		backendModeCache.Store(&cachedBackendMode{value: enabled, expiresAt: time.Now().Add(backendModeCacheTTL).UnixNano()})
		return enabled, nil
	})
	if err != nil {
		return false
	}
	enabled, ok := result.(bool)
	if !ok {
		return false
	}
	return enabled
}
