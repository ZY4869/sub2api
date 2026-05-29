package service

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"github.com/Wei-Shaw/sub2api/internal/pkg/openai"
	"go.uber.org/zap"
)

var (
	modelDateVersionSuffixPattern = regexp.MustCompile(`-(?:\d{8}|\d{4}-\d{2}-\d{2})(?:-[^-\s]+:\d+)?$`)
	openAIModelBasePattern        = regexp.MustCompile(`^(gpt-\d+(?:\.\d+)?)(?:-|$)`)
	openAIGPT54FallbackPricing    = &LiteLLMModelPricing{
		InputCostPerToken:               2.5e-06, // $2.5 per MTok
		OutputCostPerToken:              1.5e-05, // $15 per MTok
		CacheReadInputTokenCost:         2.5e-07, // $0.25 per MTok
		LongContextInputTokenThreshold:  272000,
		LongContextInputCostMultiplier:  2.0,
		LongContextOutputCostMultiplier: 1.5,
		LiteLLMProvider:                 "openai",
		Mode:                            "chat",
		SupportsPromptCaching:           true,
	}
	openAIGPT54MiniFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:       7.5e-07,
		OutputCostPerToken:      4.5e-06,
		CacheReadInputTokenCost: 7.5e-08,
		LiteLLMProvider:         "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
	openAIGPT54NanoFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:       2e-07,
		OutputCostPerToken:      1.25e-06,
		CacheReadInputTokenCost: 2e-08,
		LiteLLMProvider:         "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
	openAIGPT45PreviewFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:       7.5e-05, // $75 per MTok
		OutputCostPerToken:      1.5e-04, // $150 per MTok
		CacheReadInputTokenCost: 3.75e-05,
		LiteLLMProvider:         "openai",
		Mode:                    "chat",
		SupportsPromptCaching:   true,
	}
	openAIGPT54ProFallbackPricing = &LiteLLMModelPricing{
		InputCostPerToken:                3e-05, // $30 per MTok
		InputTokenThreshold:              272000,
		InputCostPerTokenAboveThreshold:  6e-05,
		OutputCostPerToken:               1.8e-04, // $180 per MTok
		OutputCostPerTokenAboveThreshold: 2.7e-04,
		LongContextInputTokenThreshold:   272000,
		LongContextInputCostMultiplier:   2.0,
		LongContextOutputCostMultiplier:  1.5,
		LiteLLMProvider:                  "openai",
		Mode:                             "responses",
		SupportsPromptCaching:            true,
	}
)

// LiteLLMModelPricing LiteLLM价格数据结构
// 只保留我们需要的字段，使用指针来处理可能缺失的值

// matchOpenAIModel OpenAI 模型回退匹配策略
// 回退顺序：
// 1. gpt-5.3-codex-spark* -> gpt-5.4（固定计费兜底）
// 2. gpt-5.2-20251222 -> gpt-5.2（去掉日期版本号）
// 3. gpt-5.4* -> 业务静态兜底价
// 4. 最终回退到 DefaultTestModel (gpt-5.4)
func (s *PricingService) matchOpenAIModel(model string) *LiteLLMModelPricing {
	if strings.HasPrefix(model, "gpt-5.3-codex-spark") {
		if pricing, ok := s.pricingData["gpt-5.4"]; ok {
			logger.LegacyPrintf("service.pricing", "[Pricing][SparkBilling] %s -> %s billing", model, "gpt-5.4")
			s.logOpenAIFallbackOnce(model, "gpt-5.4", "matched")
			return pricing
		}
	}

	// 尝试的回退变体
	variants := s.generateOpenAIModelVariants(model, modelDateVersionSuffixPattern)

	for _, variant := range variants {
		if pricing, ok := s.pricingData[variant]; ok {
			s.logOpenAIFallbackOnce(model, variant, "matched")
			return pricing
		}
	}

	if strings.HasPrefix(model, "gpt-4.5") {
		s.logOpenAIFallbackOnce(model, "gpt-4.5-preview(static)", "matched")
		return openAIGPT45PreviewFallbackPricing
	}

	if strings.HasPrefix(model, "gpt-5.4-pro") {
		s.logOpenAIFallbackOnce(model, "gpt-5.4-pro(static)", "matched")
		return openAIGPT54ProFallbackPricing
	}

	if strings.HasPrefix(model, "gpt-5.4-mini") {
		s.logOpenAIFallbackOnce(model, "gpt-5.4-mini(static)", "matched")
		return openAIGPT54MiniFallbackPricing
	}

	if strings.HasPrefix(model, "gpt-5.4-nano") {
		s.logOpenAIFallbackOnce(model, "gpt-5.4-nano(static)", "matched")
		return openAIGPT54NanoFallbackPricing
	}

	if strings.HasPrefix(model, "gpt-5.4") {
		s.logOpenAIFallbackOnce(model, "gpt-5.4(static)", "matched")
		return openAIGPT54FallbackPricing
	}

	// 最终回退到 DefaultTestModel
	defaultModel := strings.ToLower(openai.DefaultTestModel)
	if pricing, ok := s.pricingData[defaultModel]; ok {
		s.logOpenAIFallbackOnce(model, defaultModel, "default")
		return pricing
	}

	return nil
}

// generateOpenAIModelVariants 生成 OpenAI 模型的回退变体列表
func (s *PricingService) logOpenAIFallbackOnce(model string, target string, kind string) {
	if s == nil {
		return
	}
	key := strings.TrimSpace(strings.ToLower(kind + "|" + model + "|" + target))
	if key == "" {
		return
	}
	if _, loaded := s.fallbackLogs.LoadOrStore(key, struct{}{}); loaded {
		return
	}
	message := "[Pricing] OpenAI fallback matched %s -> %s"
	if strings.TrimSpace(strings.ToLower(kind)) == "default" {
		message = "[Pricing] OpenAI fallback to default model %s -> %s"
	}
	logger.With(zap.String("component", "service.pricing")).Debug(fmt.Sprintf(message, model, target))
}

func (s *PricingService) generateOpenAIModelVariants(model string, datePattern *regexp.Regexp) []string {
	seen := make(map[string]bool)
	var variants []string

	addVariant := func(v string) {
		if v != model && !seen[v] {
			seen[v] = true
			variants = append(variants, v)
		}
	}

	// 1. 去掉日期版本号: gpt-5.2-20251222 -> gpt-5.2
	withoutDate := datePattern.ReplaceAllString(model, "")
	if withoutDate != model {
		addVariant(withoutDate)
	}

	// 2. 提取基础版本号: gpt-5.4-pro -> gpt-5.4
	// 只匹配纯数字版本号格式 gpt-X 或 gpt-X.Y，不匹配 gpt-4o 这种带字母后缀的
	if matches := openAIModelBasePattern.FindStringSubmatch(model); len(matches) > 1 {
		addVariant(matches[1])
	}

	// 3. 同时去掉日期后再提取基础版本号
	if withoutDate != model {
		if matches := openAIModelBasePattern.FindStringSubmatch(withoutDate); len(matches) > 1 {
			addVariant(matches[1])
		}
	}

	return variants
}

// isNumeric 检查字符串是否为纯数字
func isNumeric(s string) bool {
	for _, c := range s {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
