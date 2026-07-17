package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

const (
	defaultUpstreamBillingProbeConcurrency = 2
	defaultUpstreamBillingProbeTimeout     = 20
	maxUpstreamBillingProbeConcurrency     = 10
	maxUpstreamBillingProbeTimeout         = 120
)

type UpstreamBillingProbeSettings struct {
	Enabled          bool `json:"enabled"`
	BatchConcurrency int  `json:"batch_concurrency"`
	TimeoutSeconds   int  `json:"timeout_seconds"`
}

func DefaultUpstreamBillingProbeSettings() *UpstreamBillingProbeSettings {
	return &UpstreamBillingProbeSettings{
		Enabled:          true,
		BatchConcurrency: defaultUpstreamBillingProbeConcurrency,
		TimeoutSeconds:   defaultUpstreamBillingProbeTimeout,
	}
}

func NormalizeUpstreamBillingProbeSettings(settings *UpstreamBillingProbeSettings) *UpstreamBillingProbeSettings {
	if settings == nil {
		return DefaultUpstreamBillingProbeSettings()
	}
	out := *settings
	if out.BatchConcurrency <= 0 {
		out.BatchConcurrency = defaultUpstreamBillingProbeConcurrency
	}
	if out.BatchConcurrency > maxUpstreamBillingProbeConcurrency {
		out.BatchConcurrency = maxUpstreamBillingProbeConcurrency
	}
	if out.TimeoutSeconds <= 0 {
		out.TimeoutSeconds = defaultUpstreamBillingProbeTimeout
	}
	if out.TimeoutSeconds > maxUpstreamBillingProbeTimeout {
		out.TimeoutSeconds = maxUpstreamBillingProbeTimeout
	}
	return &out
}

func (s *SettingService) GetUpstreamBillingProbeSettings(ctx context.Context) (*UpstreamBillingProbeSettings, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultUpstreamBillingProbeSettings(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyUpstreamBillingProbeSettings)
	if err != nil {
		if err == ErrSettingNotFound {
			return DefaultUpstreamBillingProbeSettings(), nil
		}
		return nil, fmt.Errorf("get upstream billing probe settings: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return DefaultUpstreamBillingProbeSettings(), nil
	}
	var settings UpstreamBillingProbeSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, fmt.Errorf("unmarshal upstream billing probe settings: %w", err)
	}
	return NormalizeUpstreamBillingProbeSettings(&settings), nil
}

func (s *SettingService) UpdateUpstreamBillingProbeSettings(ctx context.Context, settings *UpstreamBillingProbeSettings) (*UpstreamBillingProbeSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, ErrSettingNotFound
	}
	normalized := NormalizeUpstreamBillingProbeSettings(settings)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal upstream billing probe settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyUpstreamBillingProbeSettings, string(data)); err != nil {
		return nil, err
	}
	s.notifyUpdateCallbacks()
	return normalized, nil
}
