package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

const (
	ClientIPModeGin     = "gin"
	ClientIPModeHeaders = "headers"
)

var ErrClientIPSettingsInvalid = infraerrors.BadRequest("CLIENT_IP_SETTINGS_INVALID", "client IP settings are invalid")

type ClientIPSettings struct {
	Mode        string   `json:"mode"`
	Headers     []string `json:"headers"`
	XFFHopIndex int      `json:"xff_hop_index"`
}

func DefaultClientIPSettingsFromConfig(cfg *config.Config) *ClientIPSettings {
	settings := &ClientIPSettings{
		Mode:        ClientIPModeGin,
		Headers:     []string{},
		XFFHopIndex: 0,
	}
	if cfg == nil {
		return settings
	}
	settings.Mode = cfg.Server.ClientIP.Mode
	settings.Headers = append([]string(nil), cfg.Server.ClientIP.Headers...)
	settings.XFFHopIndex = cfg.Server.ClientIP.XFFHopIndex
	return NormalizeClientIPSettings(settings)
}

func NormalizeClientIPSettings(input *ClientIPSettings) *ClientIPSettings {
	defaults := &ClientIPSettings{Mode: ClientIPModeGin, Headers: []string{}, XFFHopIndex: 0}
	if input == nil {
		return defaults
	}
	out := *input
	out.Mode = strings.ToLower(strings.TrimSpace(out.Mode))
	switch out.Mode {
	case ClientIPModeHeaders:
	default:
		out.Mode = ClientIPModeGin
	}
	if out.XFFHopIndex < 0 {
		out.XFFHopIndex = 0
	}
	out.Headers = normalizeHTTPHeaderList(out.Headers)
	return &out
}

func ValidateClientIPSettings(settings *ClientIPSettings) error {
	normalized := NormalizeClientIPSettings(settings)
	switch normalized.Mode {
	case ClientIPModeGin:
		return nil
	case ClientIPModeHeaders:
		if len(normalized.Headers) == 0 {
			return ErrClientIPSettingsInvalid.WithCause(fmt.Errorf("headers are required when mode=headers"))
		}
	default:
		return ErrClientIPSettingsInvalid.WithCause(fmt.Errorf("unsupported mode: %s", normalized.Mode))
	}
	for _, header := range normalized.Headers {
		if header == "" || strings.ContainsAny(header, " \t\r\n:") {
			return ErrClientIPSettingsInvalid.WithCause(fmt.Errorf("invalid header name: %s", header))
		}
	}
	return nil
}

func (s *SettingService) GetClientIPSettings(ctx context.Context) (*ClientIPSettings, error) {
	defaults := DefaultClientIPSettingsFromConfig(s.configOrNil())
	if s == nil || s.settingRepo == nil {
		return defaults, nil
	}
	raw, err := s.settingRepo.GetValue(ctx, SettingKeyClientIPSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return defaults, nil
		}
		return nil, fmt.Errorf("get client IP settings: %w", err)
	}
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return defaults, nil
	}
	var settings ClientIPSettings
	if err := json.Unmarshal([]byte(raw), &settings); err != nil {
		return nil, ErrClientIPSettingsInvalid.WithCause(err)
	}
	return NormalizeClientIPSettings(&settings), nil
}

func (s *SettingService) SetClientIPSettings(ctx context.Context, settings *ClientIPSettings) (*ClientIPSettings, error) {
	normalized := NormalizeClientIPSettings(settings)
	if err := ValidateClientIPSettings(normalized); err != nil {
		return nil, err
	}
	data, err := json.Marshal(normalized)
	if err != nil {
		return nil, fmt.Errorf("marshal client IP settings: %w", err)
	}
	if err := s.settingRepo.Set(ctx, SettingKeyClientIPSettings, string(data)); err != nil {
		return nil, fmt.Errorf("save client IP settings: %w", err)
	}
	s.notifyUpdateCallbacks()
	return normalized, nil
}

func (s *SettingService) configOrNil() *config.Config {
	if s == nil {
		return nil
	}
	return s.cfg
}

func normalizeHTTPHeaderList(values []string) []string {
	if len(values) == 0 {
		return []string{}
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		header := strings.TrimSpace(value)
		if header == "" {
			continue
		}
		key := strings.ToLower(header)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, header)
	}
	return out
}
