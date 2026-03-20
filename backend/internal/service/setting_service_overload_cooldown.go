package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	overloadCooldownMinMinutes = 1
	overloadCooldownMaxMinutes = 120
)

func (s *SettingService) GetOverloadCooldownSettings(ctx context.Context) (*OverloadCooldownSettings, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultOverloadCooldownSettings(), nil
	}

	value, err := s.settingRepo.GetValue(ctx, SettingKeyOverloadCooldownSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultOverloadCooldownSettings(), nil
		}
		return nil, fmt.Errorf("get overload cooldown settings: %w", err)
	}

	return decodeOverloadCooldownSettings(value), nil
}

func (s *SettingService) SetOverloadCooldownSettings(ctx context.Context, settings *OverloadCooldownSettings) error {
	if settings == nil {
		return fmt.Errorf("settings cannot be nil")
	}

	normalized, err := normalizeOverloadCooldownSettingsForSave(settings)
	if err != nil {
		return err
	}

	data, err := json.Marshal(normalized)
	if err != nil {
		return fmt.Errorf("marshal overload cooldown settings: %w", err)
	}

	if err := s.settingRepo.Set(ctx, SettingKeyOverloadCooldownSettings, string(data)); err != nil {
		return err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return nil
}

func decodeOverloadCooldownSettings(raw string) *OverloadCooldownSettings {
	defaults := DefaultOverloadCooldownSettings()
	if strings.TrimSpace(raw) == "" {
		return defaults
	}

	var settings OverloadCooldownSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return defaults
	}

	settings.CooldownMinutes = clampOverloadCooldownMinutes(settings.CooldownMinutes)
	return &settings
}

func normalizeOverloadCooldownSettingsForSave(settings *OverloadCooldownSettings) (*OverloadCooldownSettings, error) {
	normalized := *settings
	if normalized.Enabled {
		if normalized.CooldownMinutes < overloadCooldownMinMinutes || normalized.CooldownMinutes > overloadCooldownMaxMinutes {
			return nil, fmt.Errorf("cooldown_minutes must be between 1-120")
		}
		return &normalized, nil
	}

	if normalized.CooldownMinutes < overloadCooldownMinMinutes || normalized.CooldownMinutes > overloadCooldownMaxMinutes {
		normalized.CooldownMinutes = DefaultOverloadCooldownSettings().CooldownMinutes
	}
	return &normalized, nil
}

func clampOverloadCooldownMinutes(value int) int {
	if value < overloadCooldownMinMinutes {
		return overloadCooldownMinMinutes
	}
	if value > overloadCooldownMaxMinutes {
		return overloadCooldownMaxMinutes
	}
	return value
}
