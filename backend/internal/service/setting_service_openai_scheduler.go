package service

import (
	"context"
	"errors"
	"strconv"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/config"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type OpenAIAdvancedSchedulerRuntimeSettings struct {
	Enabled                     bool
	StickyWeightedEnabled       bool
	SubscriptionPriorityEnabled bool
	LBTopK                      int
	Weights                     config.GatewayOpenAIWSSchedulerScoreWeights
}

func (s *SettingService) parseOpenAIAdvancedSchedulerSettings(result *SystemSettings, settings map[string]string) {
	result.OpenAIAdvancedSchedulerEnabled = !isFalseSettingValue(settings[SettingKeyOpenAIAdvancedSchedulerEnabled])
	result.OpenAIAdvancedSchedulerStickyWeightedEnabled = !isFalseSettingValue(settings[SettingKeyOpenAIAdvancedSchedulerStickyWeightedEnabled])
	result.OpenAIAdvancedSchedulerSubscriptionPriorityEnabled = settings[SettingKeyOpenAIAdvancedSchedulerSubscriptionPriorityEnabled] == "true"
	result.OpenAIAdvancedSchedulerLBTopK = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerLBTopK])
	result.OpenAIAdvancedSchedulerWeightPriority = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightPriority])
	result.OpenAIAdvancedSchedulerWeightLoad = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightLoad])
	result.OpenAIAdvancedSchedulerWeightQueue = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightQueue])
	result.OpenAIAdvancedSchedulerWeightErrorRate = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightErrorRate])
	result.OpenAIAdvancedSchedulerWeightTTFT = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightTTFT])
	result.OpenAIAdvancedSchedulerWeightQuotaHeadroom = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightQuotaHeadroom])
	result.OpenAIAdvancedSchedulerWeightPreviousResponse = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightPreviousResponse])
	result.OpenAIAdvancedSchedulerWeightSessionSticky = strings.TrimSpace(settings[SettingKeyOpenAIAdvancedSchedulerWeightSessionSticky])

	result.OpenAIAdvancedSchedulerEffectiveLBTopK = strconv.Itoa(s.openAIAdvancedSchedulerConfigLBTopK())
	weights := s.openAIAdvancedSchedulerConfigWeights()
	result.OpenAIAdvancedSchedulerEffectiveWeightPriority = formatOpenAIAdvancedSchedulerFloat(weights.Priority)
	result.OpenAIAdvancedSchedulerEffectiveWeightLoad = formatOpenAIAdvancedSchedulerFloat(weights.Load)
	result.OpenAIAdvancedSchedulerEffectiveWeightQueue = formatOpenAIAdvancedSchedulerFloat(weights.Queue)
	result.OpenAIAdvancedSchedulerEffectiveWeightErrorRate = formatOpenAIAdvancedSchedulerFloat(weights.ErrorRate)
	result.OpenAIAdvancedSchedulerEffectiveWeightTTFT = formatOpenAIAdvancedSchedulerFloat(weights.TTFT)
	result.OpenAIAdvancedSchedulerEffectiveWeightQuotaHeadroom = formatOpenAIAdvancedSchedulerFloat(weights.QuotaHeadroom)
	result.OpenAIAdvancedSchedulerEffectiveWeightPreviousResponse = formatOpenAIAdvancedSchedulerFloat(weights.PreviousResponse)
	result.OpenAIAdvancedSchedulerEffectiveWeightSessionSticky = formatOpenAIAdvancedSchedulerFloat(weights.SessionSticky)
}

func (s *SettingService) normalizeOpenAIAdvancedSchedulerSettings(settings *SystemSettings) error {
	lbTopK, err := normalizeOptionalPositiveIntString(settings.OpenAIAdvancedSchedulerLBTopK)
	if err != nil {
		return infraerrors.BadRequest("OPENAI_ADVANCED_SCHEDULER_INVALID", "openai_advanced_scheduler_lb_top_k must be empty or a positive integer")
	}
	settings.OpenAIAdvancedSchedulerLBTopK = lbTopK

	fields := []*string{
		&settings.OpenAIAdvancedSchedulerWeightPriority,
		&settings.OpenAIAdvancedSchedulerWeightLoad,
		&settings.OpenAIAdvancedSchedulerWeightQueue,
		&settings.OpenAIAdvancedSchedulerWeightErrorRate,
		&settings.OpenAIAdvancedSchedulerWeightTTFT,
		&settings.OpenAIAdvancedSchedulerWeightQuotaHeadroom,
		&settings.OpenAIAdvancedSchedulerWeightPreviousResponse,
		&settings.OpenAIAdvancedSchedulerWeightSessionSticky,
	}
	for _, field := range fields {
		normalized, err := normalizeOptionalNonNegativeFloatString(*field)
		if err != nil {
			return infraerrors.BadRequest("OPENAI_ADVANCED_SCHEDULER_INVALID", "openai advanced scheduler weights must be empty or non-negative numbers")
		}
		*field = normalized
	}

	effective := s.openAIAdvancedSchedulerConfigWeights()
	baseSum := resolveOpenAIAdvancedSchedulerWeight(settings.OpenAIAdvancedSchedulerWeightPriority, effective.Priority) +
		resolveOpenAIAdvancedSchedulerWeight(settings.OpenAIAdvancedSchedulerWeightLoad, effective.Load) +
		resolveOpenAIAdvancedSchedulerWeight(settings.OpenAIAdvancedSchedulerWeightQueue, effective.Queue) +
		resolveOpenAIAdvancedSchedulerWeight(settings.OpenAIAdvancedSchedulerWeightErrorRate, effective.ErrorRate) +
		resolveOpenAIAdvancedSchedulerWeight(settings.OpenAIAdvancedSchedulerWeightTTFT, effective.TTFT) +
		resolveOpenAIAdvancedSchedulerWeight(settings.OpenAIAdvancedSchedulerWeightQuotaHeadroom, effective.QuotaHeadroom)
	if baseSum <= 0 {
		return infraerrors.BadRequest("OPENAI_ADVANCED_SCHEDULER_INVALID", "openai advanced scheduler base weights must not all be zero")
	}
	return nil
}

func (s *SettingService) GetOpenAIAdvancedSchedulerRuntimeSettings(ctx context.Context) OpenAIAdvancedSchedulerRuntimeSettings {
	runtime := DefaultOpenAIAdvancedSchedulerRuntimeSettings()
	runtime.LBTopK = s.openAIAdvancedSchedulerConfigLBTopK()
	runtime.Weights = s.openAIAdvancedSchedulerConfigWeights()
	if s == nil || s.settingRepo == nil {
		return runtime
	}
	keys := []string{
		SettingKeyOpenAIAdvancedSchedulerEnabled,
		SettingKeyOpenAIAdvancedSchedulerStickyWeightedEnabled,
		SettingKeyOpenAIAdvancedSchedulerSubscriptionPriorityEnabled,
		SettingKeyOpenAIAdvancedSchedulerLBTopK,
		SettingKeyOpenAIAdvancedSchedulerWeightPriority,
		SettingKeyOpenAIAdvancedSchedulerWeightLoad,
		SettingKeyOpenAIAdvancedSchedulerWeightQueue,
		SettingKeyOpenAIAdvancedSchedulerWeightErrorRate,
		SettingKeyOpenAIAdvancedSchedulerWeightTTFT,
		SettingKeyOpenAIAdvancedSchedulerWeightQuotaHeadroom,
		SettingKeyOpenAIAdvancedSchedulerWeightPreviousResponse,
		SettingKeyOpenAIAdvancedSchedulerWeightSessionSticky,
	}
	values, err := s.settingRepo.GetMultiple(ctx, keys)
	if err != nil {
		return runtime
	}
	runtime.Enabled = !isFalseSettingValue(values[SettingKeyOpenAIAdvancedSchedulerEnabled])
	runtime.StickyWeightedEnabled = !isFalseSettingValue(values[SettingKeyOpenAIAdvancedSchedulerStickyWeightedEnabled])
	runtime.SubscriptionPriorityEnabled = strings.TrimSpace(values[SettingKeyOpenAIAdvancedSchedulerSubscriptionPriorityEnabled]) == "true"
	if topK, err := normalizeOptionalPositiveIntString(values[SettingKeyOpenAIAdvancedSchedulerLBTopK]); err == nil && topK != "" {
		if parsed, parseErr := strconv.Atoi(topK); parseErr == nil && parsed > 0 {
			runtime.LBTopK = parsed
		}
	}
	runtime.Weights.Priority = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightPriority], runtime.Weights.Priority)
	runtime.Weights.Load = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightLoad], runtime.Weights.Load)
	runtime.Weights.Queue = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightQueue], runtime.Weights.Queue)
	runtime.Weights.ErrorRate = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightErrorRate], runtime.Weights.ErrorRate)
	runtime.Weights.TTFT = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightTTFT], runtime.Weights.TTFT)
	runtime.Weights.QuotaHeadroom = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightQuotaHeadroom], runtime.Weights.QuotaHeadroom)
	runtime.Weights.PreviousResponse = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightPreviousResponse], runtime.Weights.PreviousResponse)
	runtime.Weights.SessionSticky = resolveOpenAIAdvancedSchedulerWeight(values[SettingKeyOpenAIAdvancedSchedulerWeightSessionSticky], runtime.Weights.SessionSticky)
	return runtime
}

func DefaultOpenAIAdvancedSchedulerRuntimeSettings() OpenAIAdvancedSchedulerRuntimeSettings {
	return OpenAIAdvancedSchedulerRuntimeSettings{
		Enabled:                     true,
		StickyWeightedEnabled:       true,
		SubscriptionPriorityEnabled: false,
		LBTopK:                      7,
		Weights: config.GatewayOpenAIWSSchedulerScoreWeights{
			Priority:         1.0,
			Load:             1.0,
			Queue:            0.7,
			ErrorRate:        0.8,
			TTFT:             0.5,
			QuotaHeadroom:    0,
			PreviousResponse: 5.0,
			SessionSticky:    3.0,
		},
	}
}

func (s *SettingService) openAIAdvancedSchedulerConfigLBTopK() int {
	if s != nil && s.cfg != nil && s.cfg.Gateway.OpenAIWS.LBTopK > 0 {
		return s.cfg.Gateway.OpenAIWS.LBTopK
	}
	return 7
}

func (s *SettingService) openAIAdvancedSchedulerConfigWeights() config.GatewayOpenAIWSSchedulerScoreWeights {
	if s != nil && s.cfg != nil {
		weights := s.cfg.Gateway.OpenAIWS.SchedulerScoreWeights
		baseSum := weights.Priority + weights.Load + weights.Queue + weights.ErrorRate + weights.TTFT + weights.QuotaHeadroom
		if baseSum > 0 {
			return weights
		}
	}
	return config.GatewayOpenAIWSSchedulerScoreWeights{
		Priority:         1.0,
		Load:             1.0,
		Queue:            0.7,
		ErrorRate:        0.8,
		TTFT:             0.5,
		QuotaHeadroom:    0,
		PreviousResponse: 5.0,
		SessionSticky:    3.0,
	}
}

func normalizeOptionalPositiveIntString(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	value, err := strconv.Atoi(trimmed)
	if err != nil || value <= 0 {
		if err == nil {
			err = errors.New("value must be positive")
		}
		return "", err
	}
	return strconv.Itoa(value), nil
}

func normalizeOptionalNonNegativeFloatString(raw string) (string, error) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return "", nil
	}
	value, err := strconv.ParseFloat(trimmed, 64)
	if err != nil || value < 0 {
		return "", err
	}
	return formatOpenAIAdvancedSchedulerFloat(value), nil
}

func formatOpenAIAdvancedSchedulerFloat(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}

func resolveOpenAIAdvancedSchedulerWeight(normalized string, fallback float64) float64 {
	value, err := strconv.ParseFloat(strings.TrimSpace(normalized), 64)
	if err != nil || value < 0 {
		return fallback
	}
	return value
}
