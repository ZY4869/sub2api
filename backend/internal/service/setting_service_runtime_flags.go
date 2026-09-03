package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

func (s *SettingService) IsTurnstileEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyTurnstileEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) IsPasskeyEnabled(ctx context.Context) bool {
	if s == nil || s.settingRepo == nil || s.cfg == nil || !s.cfg.WebAuthn.Enabled {
		return false
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyPasskeyEnabled)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) GetTurnstileSecretKey(ctx context.Context) string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyTurnstileSecretKey)
	if err != nil {
		return ""
	}
	return value
}

func (s *SettingService) GetCaptchaRuntime(ctx context.Context) CaptchaRuntimeSettings {
	if s == nil || s.settingRepo == nil {
		return CaptchaRuntimeSettings{Provider: CaptchaProviderNone}
	}
	keys := []string{
		SettingKeyTurnstileEnabled,
		SettingKeyTurnstileSecretKey,
		SettingKeyTencentCaptchaEnabled,
		SettingKeyTencentCaptchaAppID,
		SettingKeyTencentCaptchaAppSecretKey,
		SettingKeyTencentCaptchaCloudSecretID,
		SettingKeyTencentCaptchaCloudSecretKey,
		SettingKeyAliyunCaptchaEnabled,
		SettingKeyAliyunCaptchaSceneID,
		SettingKeyAliyunCaptchaPrefix,
		SettingKeyAliyunCaptchaRegion,
		SettingKeyAliyunCaptchaAccessKeyID,
		SettingKeyAliyunCaptchaAccessKeySecret,
	}
	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return CaptchaRuntimeSettings{Provider: CaptchaProviderNone}
	}
	runtime := CaptchaRuntimeSettings{
		TurnstileEnabled:      values[SettingKeyTurnstileEnabled] == "true",
		TurnstileSecretKey:    values[SettingKeyTurnstileSecretKey],
		TencentEnabled:        values[SettingKeyTencentCaptchaEnabled] == "true",
		TencentAppID:          strings.TrimSpace(values[SettingKeyTencentCaptchaAppID]),
		TencentAppSecretKey:   strings.TrimSpace(values[SettingKeyTencentCaptchaAppSecretKey]),
		TencentCloudSecretID:  strings.TrimSpace(values[SettingKeyTencentCaptchaCloudSecretID]),
		TencentCloudSecretKey: strings.TrimSpace(values[SettingKeyTencentCaptchaCloudSecretKey]),
		AliyunEnabled:         values[SettingKeyAliyunCaptchaEnabled] == "true",
		AliyunSceneID:         strings.TrimSpace(values[SettingKeyAliyunCaptchaSceneID]),
		AliyunPrefix:          strings.TrimSpace(values[SettingKeyAliyunCaptchaPrefix]),
		AliyunRegion:          strings.TrimSpace(values[SettingKeyAliyunCaptchaRegion]),
		AliyunAccessKeyID:     strings.TrimSpace(values[SettingKeyAliyunCaptchaAccessKeyID]),
		AliyunAccessKeySecret: strings.TrimSpace(values[SettingKeyAliyunCaptchaAccessKeySecret]),
	}
	runtime.Provider = CaptchaProviderFromRuntime(runtime)
	return runtime
}

func (s *SettingService) IsIdentityPatchEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEnableIdentityPatch)
	if err != nil {
		return true
	}
	return value == "true"
}

func (s *SettingService) GetIdentityPatchPrompt(ctx context.Context) string {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyIdentityPatchPrompt)
	if err != nil {
		return ""
	}
	return value
}

func (s *SettingService) IsModelFallbackEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyEnableModelFallback)
	if err != nil {
		return false
	}
	return value == "true"
}

// IsOpenAIAlphaSearchEnabled is compatible with legacy installations: a
// missing key means enabled, while only an explicit false disables the
// endpoint. Repository failures are returned so callers do not silently
// expose a partially configured gateway.
func (s *SettingService) IsOpenAIAlphaSearchEnabled(ctx context.Context) (bool, error) {
	if s == nil || s.settingRepo == nil {
		return true, nil
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyOpenAIAlphaSearchEnabled)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return true, nil
		}
		return false, fmt.Errorf("get %s: %w", SettingKeyOpenAIAlphaSearchEnabled, err)
	}
	return !isFalseSettingValue(value), nil
}

func (s *SettingService) GetFallbackModel(ctx context.Context, platform string) string {
	var key string
	var defaultModel string
	switch platform {
	case PlatformAnthropic:
		key = SettingKeyFallbackModelAnthropic
		defaultModel = "claude-3-5-sonnet-20241022"
	case PlatformOpenAI:
		key = SettingKeyFallbackModelOpenAI
		defaultModel = "gpt-4o"
	case PlatformGemini:
		key = SettingKeyFallbackModelGemini
		defaultModel = "gemini-2.5-pro"
	case PlatformAntigravity:
		key = SettingKeyFallbackModelAntigravity
		defaultModel = "gemini-2.5-pro"
	default:
		return ""
	}
	value, err := s.settingRepo.GetValue(ctx, key)
	if err != nil || value == "" {
		return defaultModel
	}
	return value
}

func (s *SettingService) GetStreamTimeoutSettings(ctx context.Context) (*StreamTimeoutSettings, error) {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyStreamTimeoutSettings)
	if err != nil {
		if errors.Is(err, ErrSettingNotFound) {
			return DefaultStreamTimeoutSettings(), nil
		}
		return nil, fmt.Errorf("get stream timeout settings: %w", err)
	}
	if value == "" {
		return DefaultStreamTimeoutSettings(), nil
	}
	var settings StreamTimeoutSettings
	if err := json.Unmarshal([]byte(value), &settings); err != nil {
		return DefaultStreamTimeoutSettings(), nil
	}
	if settings.TempUnschedMinutes < 1 {
		settings.TempUnschedMinutes = 1
	}
	if settings.TempUnschedMinutes > 60 {
		settings.TempUnschedMinutes = 60
	}
	if settings.ThresholdCount < 1 {
		settings.ThresholdCount = 1
	}
	if settings.ThresholdCount > 10 {
		settings.ThresholdCount = 10
	}
	if settings.ThresholdWindowMinutes < 1 {
		settings.ThresholdWindowMinutes = 1
	}
	if settings.ThresholdWindowMinutes > 60 {
		settings.ThresholdWindowMinutes = 60
	}
	switch settings.Action {
	case StreamTimeoutActionTempUnsched, StreamTimeoutActionError, StreamTimeoutActionNone:
	default:
		settings.Action = StreamTimeoutActionTempUnsched
	}
	return &settings, nil
}

func (s *SettingService) IsUngroupedKeySchedulingAllowed(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAllowUngroupedKeyScheduling)
	if err != nil {
		return false
	}
	return value == "true"
}

func (s *SettingService) IsMultiGroupRoutingEnabled(ctx context.Context) bool {
	value, err := s.settingRepo.GetValue(ctx, SettingKeyMultiGroupRoutingEnabled)
	if err != nil {
		return errors.Is(err, ErrSettingNotFound)
	}
	return !isFalseSettingValue(value)
}

func (s *SettingService) GetAuditLogRetentionDays(ctx context.Context) int {
	if s == nil || s.settingRepo == nil {
		return DefaultAuditLogRetentionDays
	}
	value, err := s.settingRepo.GetValue(ctx, SettingKeyAuditLogRetentionDays)
	if err != nil {
		return DefaultAuditLogRetentionDays
	}
	days, err := strconv.Atoi(strings.TrimSpace(value))
	if err != nil {
		return DefaultAuditLogRetentionDays
	}
	return NormalizeAuditLogRetentionDays(days)
}
