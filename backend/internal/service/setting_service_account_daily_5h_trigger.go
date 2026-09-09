package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
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
	if parsed, err := time.Parse("15:04", normalized.TriggerTime); err != nil || parsed.Format("15:04") != normalized.TriggerTime {
		return nil, infraerrors.BadRequest("INVALID_DAILY_5H_TRIGGER_TIME", "trigger_time must be HH:mm (00:00–23:59), in Asia/Shanghai")
	}
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
