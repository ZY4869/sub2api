package service

import "strings"

// initFallbackPricing initializes hardcoded fallback pricing.
// Price unit: USD per token, aligned with LiteLLM semantics.
func (s *BillingService) initFallbackPricing() {
	// Claude 4.5 Opus
	s.fallbackPrices["claude-opus-4.5"] = &ModelPricing{
		InputPricePerToken:         5e-6,    // $5 per MTok
		OutputPricePerToken:        25e-6,   // $25 per MTok
		CacheCreationPricePerToken: 6.25e-6, // $6.25 per MTok
		CacheReadPricePerToken:     0.5e-6,  // $0.50 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 4 Sonnet
	s.fallbackPrices["claude-sonnet-4"] = &ModelPricing{
		InputPricePerToken:         3e-6,    // $3 per MTok
		OutputPricePerToken:        15e-6,   // $15 per MTok
		CacheCreationPricePerToken: 3.75e-6, // $3.75 per MTok
		CacheReadPricePerToken:     0.3e-6,  // $0.30 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3.5 Sonnet
	s.fallbackPrices["claude-3-5-sonnet"] = &ModelPricing{
		InputPricePerToken:         3e-6,    // $3 per MTok
		OutputPricePerToken:        15e-6,   // $15 per MTok
		CacheCreationPricePerToken: 3.75e-6, // $3.75 per MTok
		CacheReadPricePerToken:     0.3e-6,  // $0.30 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3.5 Haiku
	s.fallbackPrices["claude-3-5-haiku"] = &ModelPricing{
		InputPricePerToken:         1e-6,    // $1 per MTok
		OutputPricePerToken:        5e-6,    // $5 per MTok
		CacheCreationPricePerToken: 1.25e-6, // $1.25 per MTok
		CacheReadPricePerToken:     0.1e-6,  // $0.10 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3 Opus
	s.fallbackPrices["claude-3-opus"] = &ModelPricing{
		InputPricePerToken:         15e-6,    // $15 per MTok
		OutputPricePerToken:        75e-6,    // $75 per MTok
		CacheCreationPricePerToken: 18.75e-6, // $18.75 per MTok
		CacheReadPricePerToken:     1.5e-6,   // $1.50 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 3 Haiku
	s.fallbackPrices["claude-3-haiku"] = &ModelPricing{
		InputPricePerToken:         0.25e-6, // $0.25 per MTok
		OutputPricePerToken:        1.25e-6, // $1.25 per MTok
		CacheCreationPricePerToken: 0.3e-6,  // $0.30 per MTok
		CacheReadPricePerToken:     0.03e-6, // $0.03 per MTok
		SupportsCacheBreakdown:     false,
	}

	// Claude 4.6 Opus (currently priced the same as 4.5)
	s.fallbackPrices["claude-opus-4.6"] = s.fallbackPrices["claude-opus-4.5"]

	// Claude 4.7 Opus (暂与4.6同价，待官方定价更新)
	s.fallbackPrices["claude-opus-4.7"] = s.fallbackPrices["claude-opus-4.6"]

	// Gemini 3.1 Pro
	s.fallbackPrices["gemini-3.1-pro"] = &ModelPricing{
		InputPricePerToken:         2e-6,   // $2 per MTok
		OutputPricePerToken:        12e-6,  // $12 per MTok
		CacheCreationPricePerToken: 2e-6,   // $2 per MTok
		CacheReadPricePerToken:     0.2e-6, // $0.20 per MTok
		SupportsCacheBreakdown:     false,
	}

	// OpenAI GPT-5.4 business baseline pricing.
	s.fallbackPrices["gpt-5.4"] = &ModelPricing{
		InputPricePerToken:             2.5e-6,  // $2.5 per MTok
		InputPricePerTokenPriority:     5e-6,    // $5 per MTok
		OutputPricePerToken:            15e-6,   // $15 per MTok
		OutputPricePerTokenPriority:    30e-6,   // $30 per MTok
		CacheCreationPricePerToken:     2.5e-6,  // $2.5 per MTok
		CacheReadPricePerToken:         0.25e-6, // $0.25 per MTok
		CacheReadPricePerTokenPriority: 0.5e-6,  // $0.5 per MTok
		SupportsCacheBreakdown:         false,
		LongContextInputThreshold:      openAIGPT54LongContextInputThreshold,
		LongContextInputMultiplier:     openAIGPT54LongContextInputMultiplier,
		LongContextOutputMultiplier:    openAIGPT54LongContextOutputMultiplier,
	}
	// OpenAI GPT-5.4 mini/nano/pro official fallback pricing.
	s.fallbackPrices["gpt-5.4-mini"] = &ModelPricing{
		InputPricePerToken:     7.5e-7,
		OutputPricePerToken:    4.5e-6,
		CacheReadPricePerToken: 7.5e-8,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["gpt-5.4-nano"] = &ModelPricing{
		InputPricePerToken:     2e-7,
		OutputPricePerToken:    1.25e-6,
		CacheReadPricePerToken: 2e-8,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["gpt-5.4-pro"] = &ModelPricing{
		InputPricePerToken:                3e-5, // $30 per MTok
		InputTokenThreshold:               openAIGPT54LongContextInputThreshold,
		InputPricePerTokenAboveThreshold:  6e-5,
		OutputPricePerToken:               1.8e-4, // $180 per MTok
		OutputTokenThreshold:              openAIGPT54LongContextInputThreshold,
		OutputPricePerTokenAboveThreshold: 2.7e-4,
		SupportsCacheBreakdown:            false,
		LongContextInputThreshold:         openAIGPT54LongContextInputThreshold,
		LongContextInputMultiplier:        openAIGPT54LongContextInputMultiplier,
		LongContextOutputMultiplier:       openAIGPT54LongContextOutputMultiplier,
	}
	// OpenAI GPT-5.2 local fallback pricing.
	s.fallbackPrices["gpt-5.2"] = &ModelPricing{
		InputPricePerToken:             1.75e-6,
		InputPricePerTokenPriority:     3.5e-6,
		OutputPricePerToken:            14e-6,
		OutputPricePerTokenPriority:    28e-6,
		CacheCreationPricePerToken:     1.75e-6,
		CacheReadPricePerToken:         0.175e-6,
		CacheReadPricePerTokenPriority: 0.35e-6,
		SupportsCacheBreakdown:         false,
	}
	// Codex Spark fallback pricing uses GPT-5.4 as the baseline (billing-only fallback).
	s.fallbackPrices["gpt-5.3-codex-spark"] = s.fallbackPrices["gpt-5.4"]

	// Chinese provider fallback pricing. Dynamic pricing remains preferred; these
	// values keep billing available when the external pricing catalog is absent.
	s.fallbackPrices["deepseek-v4-flash"] = &ModelPricing{
		InputPricePerToken:     1.4e-7,
		OutputPricePerToken:    2.8e-7,
		CacheReadPricePerToken: 2.8e-9,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["deepseek-v4-pro"] = &ModelPricing{
		InputPricePerToken:     4.35e-7,
		OutputPricePerToken:    8.7e-7,
		CacheReadPricePerToken: 3.625e-9,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["deepseek-chat"] = s.fallbackPrices["deepseek-v4-flash"]
	s.fallbackPrices["deepseek-reasoner"] = s.fallbackPrices["deepseek-v4-flash"]
	s.fallbackPrices["doubao"] = &ModelPricing{
		InputPricePerToken:     8e-7,
		OutputPricePerToken:    2e-6,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["kimi"] = &ModelPricing{
		InputPricePerToken:     2e-6,
		OutputPricePerToken:    1e-5,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["minimax"] = &ModelPricing{
		InputPricePerToken:     1e-6,
		OutputPricePerToken:    8e-6,
		SupportsCacheBreakdown: false,
	}
	s.fallbackPrices["glm"] = &ModelPricing{
		InputPricePerToken:     5e-7,
		OutputPricePerToken:    5e-7,
		SupportsCacheBreakdown: false,
	}
}

// getFallbackPricing returns fallback pricing by model family.
func (s *BillingService) getFallbackPricing(model string) *ModelPricing {
	modelLower := strings.ToLower(model)

	// Match by model family.
	if strings.Contains(modelLower, "opus") {
		if strings.Contains(modelLower, "4.7") || strings.Contains(modelLower, "4-7") {
			return s.fallbackPrices["claude-opus-4.7"]
		}
		if strings.Contains(modelLower, "4.6") || strings.Contains(modelLower, "4-6") {
			return s.fallbackPrices["claude-opus-4.6"]
		}
		if strings.Contains(modelLower, "4.5") || strings.Contains(modelLower, "4-5") {
			return s.fallbackPrices["claude-opus-4.5"]
		}
		return s.fallbackPrices["claude-3-opus"]
	}
	if strings.Contains(modelLower, "sonnet") {
		if strings.Contains(modelLower, "4") && !strings.Contains(modelLower, "3") {
			return s.fallbackPrices["claude-sonnet-4"]
		}
		return s.fallbackPrices["claude-3-5-sonnet"]
	}
	if strings.Contains(modelLower, "haiku") {
		if strings.Contains(modelLower, "3-5") || strings.Contains(modelLower, "3.5") {
			return s.fallbackPrices["claude-3-5-haiku"]
		}
		return s.fallbackPrices["claude-3-haiku"]
	}
	// Unknown Claude models fall back to Sonnet to avoid billing interruptions.
	if strings.Contains(modelLower, "claude") {
		return s.fallbackPrices["claude-sonnet-4"]
	}
	if strings.Contains(modelLower, "gemini-3.1-pro") || strings.Contains(modelLower, "gemini-3-1-pro") {
		return s.fallbackPrices["gemini-3.1-pro"]
	}
	switch {
	case strings.Contains(modelLower, "deepseek-v4-pro"):
		return s.fallbackPrices["deepseek-v4-pro"]
	case strings.Contains(modelLower, "deepseek-v4-flash"),
		strings.Contains(modelLower, "deepseek-chat"),
		strings.Contains(modelLower, "deepseek-reasoner"):
		return s.fallbackPrices["deepseek-v4-flash"]
	case strings.Contains(modelLower, "doubao"):
		return s.fallbackPrices["doubao"]
	case strings.Contains(modelLower, "kimi"):
		return s.fallbackPrices["kimi"]
	case strings.Contains(modelLower, "minimax"),
		strings.Contains(modelLower, "abab"):
		return s.fallbackPrices["minimax"]
	case strings.Contains(modelLower, "glm"),
		strings.Contains(modelLower, "chatglm"),
		strings.Contains(modelLower, "zhipu"):
		return s.fallbackPrices["glm"]
	}

	// Only match known GPT-5/Codex families to avoid mispricing unknown OpenAI models.
	if strings.Contains(modelLower, "gpt-5") || strings.Contains(modelLower, "codex") {
		normalized := normalizeCodexModel(modelLower)
		switch {
		case strings.HasPrefix(normalized, "gpt-5.4-pro"):
			return s.fallbackPrices["gpt-5.4-pro"]
		case strings.HasPrefix(normalized, "gpt-5.4-mini"):
			return s.fallbackPrices["gpt-5.4-mini"]
		case strings.HasPrefix(normalized, "gpt-5.4-nano"):
			return s.fallbackPrices["gpt-5.4-nano"]
		case strings.HasPrefix(normalized, "gpt-5.4"):
			return s.fallbackPrices["gpt-5.4"]
		case strings.HasPrefix(normalized, "gpt-5.3-codex-spark"):
			return s.fallbackPrices["gpt-5.3-codex-spark"]
		case strings.HasPrefix(normalized, "gpt-5.2"):
			return s.fallbackPrices["gpt-5.2"]
		case strings.HasPrefix(normalized, "gpt-5-pro"):
			return s.fallbackPrices["gpt-5.4"]
		}
	}

	return nil
}

// ListSupportedModels returns models covered by fallback pricing.
func (s *BillingService) ListSupportedModels() []string {
	models := make([]string, 0)
	// Return model families covered by fallback pricing.
	for model := range s.fallbackPrices {
		models = append(models, model)
	}
	return models
}

// IsModelSupported checks whether a model is covered by fallback pricing rules.
func (s *BillingService) IsModelSupported(model string) bool {
	// All Claude-family models have fallback pricing coverage.
	modelLower := strings.ToLower(model)
	return strings.Contains(modelLower, "claude") ||
		strings.Contains(modelLower, "opus") ||
		strings.Contains(modelLower, "sonnet") ||
		strings.Contains(modelLower, "haiku")
}
