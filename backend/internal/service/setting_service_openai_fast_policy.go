package service

import (
	"context"
	"errors"
	"strings"
)

func (s *SettingService) GetOpenAIFastPolicySettings(ctx context.Context) (*OpenAIFastPolicySettings, error) {
	if s == nil || s.settingRepo == nil {
		return DefaultOpenAIFastPolicySettings(), nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIFastPolicySettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultOpenAIFastPolicySettings(), nil
		}
		// Fail-open with defaults to avoid breaking gateway traffic on settings read errors.
		return DefaultOpenAIFastPolicySettings(), nil
	}
	return ParseOpenAIFastPolicySettings(value), nil
}

func (s *SettingService) IsAnthropicCacheTTL1hInjectionEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil {
		return false
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEnableAnthropicCacheTTL1hInjection)
	if err != nil {
		return false
	}
	return strings.TrimSpace(value) == "true"
}
