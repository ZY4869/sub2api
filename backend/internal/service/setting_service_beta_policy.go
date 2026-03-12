package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

func (s *SettingService) GetBetaPolicySettings(ctx context.Context) (*BetaPolicySettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyBetaPolicySettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultBetaPolicySettings(), nil
		}
		return nil, fmt.Errorf("get beta policy settings: %w", err)
	}
	if value == "" {
		return DefaultBetaPolicySettings(), nil
	}
	var settings BetaPolicySettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultBetaPolicySettings(), nil
	}
	return &settings, nil
}
