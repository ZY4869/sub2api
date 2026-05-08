package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *SettingService) GetAccountDaily5HTriggerSettings(ctx context.Context) (*AccountDaily5HTriggerSettings, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyAccountDaily5HTriggerSettings)
	if err != nil {
		if err == ErrSettingNotFound {
			return DefaultAccountDaily5HTriggerSettings(), nil
		}
		return nil, fmt.Errorf("get account daily 5h trigger settings: %w", err)
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultAccountDaily5HTriggerSettings(), nil
	}
	var settings AccountDaily5HTriggerSettings
	if err := json.Unmarshal([]byte(trimmed), &settings); err != nil {
		return nil, fmt.Errorf("unmarshal account daily 5h trigger settings: %w", err)
	}
	return NormalizeAccountDaily5HTriggerSettings(&settings), nil
}

func (s *SettingService) UpdateAccountDaily5HTriggerSettings(ctx context.Context, settings *AccountDaily5HTriggerSettings) (*AccountDaily5HTriggerSettings, error) {
	normalized := NormalizeAccountDaily5HTriggerSettings(settings)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal account daily 5h trigger settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyAccountDaily5HTriggerSettings, string(data)); err != nil {
		return nil, err
	}
	s.notifyUpdateCallbacks()
	return normalized, nil
}
