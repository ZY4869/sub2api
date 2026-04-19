package service

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"time"

	"golang.org/x/sync/singleflight"
)

type cachedMaintenanceMode struct {
	value     bool
	expiresAt int64
}

var maintenanceModeCache atomic.Value
var maintenanceModeSF singleflight.Group

const maintenanceModeCacheTTL = 30 * time.Second

func (s *SettingService) IsMaintenanceModeEnabled(ctx context.Context) bool {
	if cached, ok := maintenanceModeCache.Load().(*cachedMaintenanceMode); ok && cached != nil {
		if time.Now().UnixNano() < cached.expiresAt {
			return cached.value
		}
	}

	result, err, _ := maintenanceModeSF.Do("maintenance_mode_enabled", func() (any, error) {
		if cached, ok := maintenanceModeCache.Load().(*cachedMaintenanceMode); ok && cached != nil {
			if time.Now().UnixNano() < cached.expiresAt {
				return cached.value, nil
			}
		}
		if s == nil || s.settingRepo == nil {
			maintenanceModeCache.Store(&cachedMaintenanceMode{value: false, expiresAt: time.Now().Add(maintenanceModeCacheTTL).UnixNano()})
			return false, nil
		}
		value, readErr := s.settingRepo.GetValue(ctx, SettingKeyMaintenanceModeEnabled)
		if readErr != nil {
			if errors.Is(readErr, ErrSettingNotFound) {
				maintenanceModeCache.Store(&cachedMaintenanceMode{value: false, expiresAt: time.Now().Add(maintenanceModeCacheTTL).UnixNano()})
				return false, nil
			}
			maintenanceModeCache.Store(&cachedMaintenanceMode{value: false, expiresAt: time.Now().Add(maintenanceModeCacheTTL).UnixNano()})
			return false, nil
		}
		enabled := strings.TrimSpace(value) == "true"
		maintenanceModeCache.Store(&cachedMaintenanceMode{value: enabled, expiresAt: time.Now().Add(maintenanceModeCacheTTL).UnixNano()})
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
