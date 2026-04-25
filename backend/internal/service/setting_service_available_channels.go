package service

import (
	"context"
)

type AvailableChannelsRuntimeSettings struct {
	Enabled bool
}

func (s *SettingService) GetAvailableChannelsRuntime(ctx context.Context) AvailableChannelsRuntimeSettings {
	if s == nil || s.settingRepo == nil {
		return AvailableChannelsRuntimeSettings{Enabled: false}
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAvailableChannelsEnabled)
	if err != nil {
		return AvailableChannelsRuntimeSettings{Enabled: false}
	}
	return AvailableChannelsRuntimeSettings{Enabled: value == "true"}
}
