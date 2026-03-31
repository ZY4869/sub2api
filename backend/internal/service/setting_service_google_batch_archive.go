package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (s *SettingService) GetGoogleBatchArchiveSettings(ctx context.Context) (*GoogleBatchArchiveSettings, error) {
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyGoogleBatchArchiveSettings)
	if err != nil {
		if err == ErrSettingNotFound {
			return DefaultGoogleBatchArchiveSettings(), nil
		}
		return nil, fmt.Errorf("get google batch archive settings: %w", err)
	}
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return DefaultGoogleBatchArchiveSettings(), nil
	}
	var settings GoogleBatchArchiveSettings
	if err := json.Unmarshal([]byte(trimmed), &settings); err != nil {
		return nil, fmt.Errorf("unmarshal google batch archive settings: %w", err)
	}
	return NormalizeGoogleBatchArchiveSettings(&settings), nil
}

func (s *SettingService) UpdateGoogleBatchArchiveSettings(ctx context.Context, settings *GoogleBatchArchiveSettings) (*GoogleBatchArchiveSettings, error) {
	normalized := NormalizeGoogleBatchArchiveSettings(settings)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal google batch archive settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyGoogleBatchArchiveSettings, string(data)); err != nil {
		return nil, err
	}
	if s.onUpdate != nil {
		s.onUpdate()
	}
	return normalized, nil
}
