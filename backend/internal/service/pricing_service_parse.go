package service

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
)

func (s *PricingService) parsePricingData(body []byte) (map[string]*LiteLLMModelPricing, error) {
	// 首先解析为 map[string]json.RawMessage
	var rawData map[string]json.RawMessage
	if err := json.Unmarshal(body, &rawData); err != nil {
		return nil, fmt.Errorf("parse raw JSON: %w", err)
	}

	result := make(map[string]*LiteLLMModelPricing)
	skipped := 0

	for modelName, rawEntry := range rawData {
		// 跳过 sample_spec 等文档条目
		if modelName == "sample_spec" {
			continue
		}

		// 尝试解析每个条目
		var entry LiteLLMRawEntry
		if err := json.Unmarshal(rawEntry, &entry); err != nil {
			skipped++
			continue
		}

		// 只保留有有效价格的条目
		if !hasAnyPricingValue(entry) {
			continue
		}

		pricing := &LiteLLMModelPricing{
			Currency:              normalizeBillingCurrency(entry.Currency),
			FXRateDate:            strings.TrimSpace(entry.FXRateDate),
			FXLockedAt:            cloneBillingTime(entry.FXLockedAt),
			LiteLLMProvider:       entry.LiteLLMProvider,
			Mode:                  entry.Mode,
			SupportsPromptCaching: entry.SupportsPromptCaching,
			SupportsServiceTier:   entry.SupportsServiceTier,
		}
		if entry.USDToCNYRate != nil {
			pricing.USDToCNYRate = *entry.USDToCNYRate
		}

		if entry.InputCostPerToken != nil {
			pricing.InputCostPerToken = *entry.InputCostPerToken
		}
		if entry.InputCostPerTokenPriority != nil {
			pricing.InputCostPerTokenPriority = *entry.InputCostPerTokenPriority
		}
		if entry.InputTokenThreshold != nil {
			pricing.InputTokenThreshold = *entry.InputTokenThreshold
		} else if entry.InputCostPerTokenAbove200kTokens != nil || entry.InputCostPerTokenPriorityAbove200kTokens != nil {
			pricing.InputTokenThreshold = 200000
		}
		if entry.InputCostPerTokenAboveThreshold != nil {
			pricing.InputCostPerTokenAboveThreshold = *entry.InputCostPerTokenAboveThreshold
		} else if entry.InputCostPerTokenAbove200kTokens != nil {
			pricing.InputCostPerTokenAboveThreshold = *entry.InputCostPerTokenAbove200kTokens
		}
		if entry.InputCostPerTokenPriorityAboveThreshold != nil {
			pricing.InputCostPerTokenPriorityAboveThreshold = *entry.InputCostPerTokenPriorityAboveThreshold
		} else if entry.InputCostPerTokenPriorityAbove200kTokens != nil {
			pricing.InputCostPerTokenPriorityAboveThreshold = *entry.InputCostPerTokenPriorityAbove200kTokens
		}
		if entry.OutputCostPerToken != nil {
			pricing.OutputCostPerToken = *entry.OutputCostPerToken
		}
		if entry.OutputCostPerTokenPriority != nil {
			pricing.OutputCostPerTokenPriority = *entry.OutputCostPerTokenPriority
		}
		if entry.OutputTokenThreshold != nil {
			pricing.OutputTokenThreshold = *entry.OutputTokenThreshold
		} else if entry.OutputCostPerTokenAbove200kTokens != nil || entry.OutputCostPerTokenPriorityAbove200kTokens != nil {
			pricing.OutputTokenThreshold = 200000
		}
		if entry.OutputCostPerTokenAboveThreshold != nil {
			pricing.OutputCostPerTokenAboveThreshold = *entry.OutputCostPerTokenAboveThreshold
		} else if entry.OutputCostPerTokenAbove200kTokens != nil {
			pricing.OutputCostPerTokenAboveThreshold = *entry.OutputCostPerTokenAbove200kTokens
		}
		if entry.OutputCostPerTokenPriorityAboveThreshold != nil {
			pricing.OutputCostPerTokenPriorityAboveThreshold = *entry.OutputCostPerTokenPriorityAboveThreshold
		} else if entry.OutputCostPerTokenPriorityAbove200kTokens != nil {
			pricing.OutputCostPerTokenPriorityAboveThreshold = *entry.OutputCostPerTokenPriorityAbove200kTokens
		}
		if entry.CacheCreationInputTokenCost != nil {
			pricing.CacheCreationInputTokenCost = *entry.CacheCreationInputTokenCost
		}
		if entry.CacheCreationInputTokenCostAbove1hr != nil {
			pricing.CacheCreationInputTokenCostAbove1hr = *entry.CacheCreationInputTokenCostAbove1hr
		}
		if entry.CacheReadInputTokenCost != nil {
			pricing.CacheReadInputTokenCost = *entry.CacheReadInputTokenCost
		}
		if entry.CacheReadInputTokenCostPriority != nil {
			pricing.CacheReadInputTokenCostPriority = *entry.CacheReadInputTokenCostPriority
		}
		if entry.LongContextInputTokenThreshold != nil {
			pricing.LongContextInputTokenThreshold = *entry.LongContextInputTokenThreshold
		}
		if entry.LongContextInputCostMultiplier != nil {
			pricing.LongContextInputCostMultiplier = *entry.LongContextInputCostMultiplier
		}
		if entry.LongContextOutputCostMultiplier != nil {
			pricing.LongContextOutputCostMultiplier = *entry.LongContextOutputCostMultiplier
		}
		if entry.OutputCostPerImage != nil {
			pricing.OutputCostPerImage = *entry.OutputCostPerImage
		}
		if entry.OutputCostPerImagePriority != nil {
			pricing.OutputCostPerImagePriority = *entry.OutputCostPerImagePriority
		}
		if entry.OutputCostPerVideoRequest != nil {
			pricing.OutputCostPerVideoRequest = *entry.OutputCostPerVideoRequest
		}

		result[modelName] = pricing
	}

	if skipped > 0 {
		logger.LegacyPrintf("service.pricing", "[Pricing] Skipped %d invalid entries", skipped)
	}

	if len(result) == 0 {
		return nil, fmt.Errorf("no valid pricing entries found")
	}

	return result, nil
}

func hasAnyPricingValue(entry LiteLLMRawEntry) bool {
	return entry.InputCostPerToken != nil ||
		entry.InputCostPerTokenPriority != nil ||
		entry.InputTokenThreshold != nil ||
		entry.InputCostPerTokenAboveThreshold != nil ||
		entry.InputCostPerTokenAbove200kTokens != nil ||
		entry.InputCostPerTokenPriorityAboveThreshold != nil ||
		entry.InputCostPerTokenPriorityAbove200kTokens != nil ||
		entry.OutputCostPerToken != nil ||
		entry.OutputCostPerTokenPriority != nil ||
		entry.OutputTokenThreshold != nil ||
		entry.OutputCostPerTokenAboveThreshold != nil ||
		entry.OutputCostPerTokenAbove200kTokens != nil ||
		entry.OutputCostPerTokenPriorityAboveThreshold != nil ||
		entry.OutputCostPerTokenPriorityAbove200kTokens != nil ||
		entry.CacheCreationInputTokenCost != nil ||
		entry.CacheCreationInputTokenCostAbove1hr != nil ||
		entry.CacheReadInputTokenCost != nil ||
		entry.CacheReadInputTokenCostPriority != nil ||
		entry.OutputCostPerImage != nil ||
		entry.OutputCostPerImagePriority != nil ||
		entry.OutputCostPerVideoRequest != nil
}
