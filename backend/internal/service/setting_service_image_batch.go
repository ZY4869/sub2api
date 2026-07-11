package service

import (
	"context"
	"errors"
	"strings"
)

func (s *SettingService) IsImageBatchEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyImageBatchEnabled)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return false
		}
		return false
	}
	return strings.EqualFold(strings.TrimSpace(raw), "true")
}

func (s *SettingService) SetImageBatchEnabled(ctx context.Context, enabled bool) error {
	if s == nil || s.settingRepo == nil {
		return ErrSettingNotFound
	}
	value := "false"
	if enabled {
		value = "true"
	}
	if err := s.settingRepo.Set(ctx, SettingKeyImageBatchEnabled, value); err != nil {
		return err
	}
	s.notifyUpdateCallbacks()
	return nil
}
