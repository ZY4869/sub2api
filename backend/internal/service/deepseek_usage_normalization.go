package service

import "strings"

type usageDisplayTokens struct {
	InputTokens           int
	OutputTokens          int
	CacheCreationTokens   int
	CacheReadTokens       int
	CacheCreation5mTokens int
	CacheCreation1hTokens int
}

type deepSeekUsageNormalization struct {
	DisplayTokens usageDisplayTokens
	BillingTokens UsageTokens
}

func normalizeOpenAIUsageForDisplayAndBilling(provider string, usage OpenAIUsage) deepSeekUsageNormalization {
	hit := clampNonNegativeInt(usage.CacheReadInputTokens)
	miss := clampNonNegativeInt(usage.CacheCreationInputTokens)
	totalInput := clampNonNegativeInt(usage.InputTokens)
	nonCacheInput := totalInput - hit
	if nonCacheInput < 0 {
		nonCacheInput = 0
	}

	display := usageDisplayTokens{
		InputTokens:           nonCacheInput,
		OutputTokens:          usage.OutputTokens,
		CacheCreationTokens:   miss,
		CacheReadTokens:       hit,
		CacheCreation5mTokens: 0,
		CacheCreation1hTokens: 0,
	}
	billing := UsageTokens{
		InputTokens:         nonCacheInput,
		OutputTokens:        usage.OutputTokens,
		CacheCreationTokens: miss,
		CacheReadTokens:     hit,
	}
	if !isDeepSeekUsageProvider(provider) {
		return deepSeekUsageNormalization{
			DisplayTokens: display,
			BillingTokens: billing,
		}
	}

	display.InputTokens = nonCacheInput - miss
	if display.InputTokens < 0 {
		display.InputTokens = 0
	}
	display.CacheCreationTokens = miss
	display.CacheReadTokens = hit

	billing.InputTokens = nonCacheInput
	billing.CacheCreationTokens = 0
	billing.CacheReadTokens = hit

	return deepSeekUsageNormalization{
		DisplayTokens: display,
		BillingTokens: billing,
	}
}

func normalizeClaudeUsageForDisplayAndBilling(provider string, usage ClaudeUsage) deepSeekUsageNormalization {
	display := usageDisplayTokens{
		InputTokens:           usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		CacheCreationTokens:   usage.CacheCreationInputTokens,
		CacheReadTokens:       usage.CacheReadInputTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}
	billing := UsageTokens{
		InputTokens:           usage.InputTokens,
		OutputTokens:          usage.OutputTokens,
		CacheCreationTokens:   usage.CacheCreationInputTokens,
		CacheReadTokens:       usage.CacheReadInputTokens,
		CacheCreation5mTokens: usage.CacheCreation5mTokens,
		CacheCreation1hTokens: usage.CacheCreation1hTokens,
	}
	if !isDeepSeekUsageProvider(provider) {
		return deepSeekUsageNormalization{
			DisplayTokens: display,
			BillingTokens: billing,
		}
	}

	hit := clampNonNegativeInt(usage.CacheReadInputTokens)
	miss := clampNonNegativeInt(usage.CacheCreationInputTokens)
	totalInput := clampNonNegativeInt(usage.InputTokens)
	nonCacheInput := totalInput - hit - miss
	if nonCacheInput < 0 {
		nonCacheInput = 0
	}

	display.InputTokens = nonCacheInput
	display.CacheCreationTokens = miss
	display.CacheReadTokens = hit
	display.CacheCreation5mTokens = 0
	display.CacheCreation1hTokens = 0

	billing.InputTokens = nonCacheInput + miss
	billing.CacheCreationTokens = 0
	billing.CacheReadTokens = hit
	billing.CacheCreation5mTokens = 0
	billing.CacheCreation1hTokens = 0

	return deepSeekUsageNormalization{
		DisplayTokens: display,
		BillingTokens: billing,
	}
}

func isDeepSeekUsageProvider(provider string) bool {
	return strings.TrimSpace(strings.ToLower(provider)) == PlatformDeepSeek
}

func clampNonNegativeInt(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
