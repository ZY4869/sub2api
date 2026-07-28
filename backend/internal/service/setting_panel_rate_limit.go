package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

const (
	defaultPanelRateLimitUserRPM     = 240
	defaultPanelRateLimitHeavyRPM    = 60
	defaultPanelRateLimitPublicIPRPM = 300
	maxPanelRateLimitRPM             = 100000
)

type PanelRateLimitSettings struct {
	Enabled     bool `json:"enabled"`
	UserRPM     int  `json:"user_rpm"`
	HeavyRPM    int  `json:"heavy_rpm"`
	PublicIPRPM int  `json:"public_ip_rpm"`
	ExemptAdmin bool `json:"exempt_admin"`
}

func DefaultPanelRateLimitSettings() *PanelRateLimitSettings {
	return &PanelRateLimitSettings{
		Enabled:     false,
		UserRPM:     defaultPanelRateLimitUserRPM,
		HeavyRPM:    defaultPanelRateLimitHeavyRPM,
		PublicIPRPM: defaultPanelRateLimitPublicIPRPM,
		ExemptAdmin: true,
	}
}

func NormalizePanelRateLimitSettings(settings *PanelRateLimitSettings) *PanelRateLimitSettings {
	defaults := DefaultPanelRateLimitSettings()
	if settings == nil {
		return defaults
	}
	out := *settings
	if out.UserRPM <= 0 {
		out.UserRPM = defaults.UserRPM
	}
	if out.HeavyRPM <= 0 {
		out.HeavyRPM = defaults.HeavyRPM
	}
	if out.PublicIPRPM <= 0 {
		out.PublicIPRPM = defaults.PublicIPRPM
	}
	if out.UserRPM > maxPanelRateLimitRPM {
		out.UserRPM = maxPanelRateLimitRPM
	}
	if out.HeavyRPM > maxPanelRateLimitRPM {
		out.HeavyRPM = maxPanelRateLimitRPM
	}
	if out.PublicIPRPM > maxPanelRateLimitRPM {
		out.PublicIPRPM = maxPanelRateLimitRPM
	}
	return &out
}

func (s *SettingService) GetPanelRateLimitSettings(ctx context.Context) (*PanelRateLimitSettings, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultPanelRateLimitSettings(), nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyPanelRateLimitSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultPanelRateLimitSettings(), nil
		}
		return nil, fmt.Errorf("get panel rate limit settings: %w", err)
	}
	if strings.TrimSpace(raw) == "" {
		return DefaultPanelRateLimitSettings(), nil
	}
	var settings PanelRateLimitSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, fmt.Errorf("unmarshal panel rate limit settings: %w", err)
	}
	return NormalizePanelRateLimitSettings(&settings), nil
}

func (s *SettingService) SetPanelRateLimitSettings(ctx context.Context, settings *PanelRateLimitSettings) (*PanelRateLimitSettings, error) {
	if s == nil || s.settingRepo == nil {
		return nil, ErrSettingNotFound
	}
	normalized := NormalizePanelRateLimitSettings(settings)
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal panel rate limit settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyPanelRateLimitSettings, string(data)); err != nil {
		return nil, fmt.Errorf("save panel rate limit settings: %w", err)
	}
	s.notifyUpdateCallbacks()
	return normalized, nil
}
