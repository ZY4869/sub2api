package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

type ChannelMonitorRuntimeSettings struct {
	Enabled                bool
	DefaultIntervalSeconds int
}

func (s *SettingService) GetChannelMonitorRuntime(ctx context.Context) (*ChannelMonitorRuntimeSettings, error) {
	values, err := s.settingRepo.GetMultiple(ctx, []string{
		SettingKeyChannelMonitorEnabled,
		SettingKeyChannelMonitorDefaultIntervalSeconds,
	})
	if err != nil {
		return nil, fmt.Errorf("get channel monitor runtime: %w", err)
	}

	enabled := values[SettingKeyChannelMonitorEnabled] == "true"
	intervalSeconds := 60
	if raw := strings.TrimSpace(values[SettingKeyChannelMonitorDefaultIntervalSeconds]); raw != "" {
		if v, err := strconv.Atoi(raw); err == nil {
			intervalSeconds = v
		}
	}
	if intervalSeconds <= 0 {
		intervalSeconds = 60
	}
	if intervalSeconds < 15 {
		intervalSeconds = 15
	}
	if intervalSeconds > 3600 {
		intervalSeconds = 3600
	}

	return &ChannelMonitorRuntimeSettings{
		Enabled:                enabled,
		DefaultIntervalSeconds: intervalSeconds,
	}, nil
}
