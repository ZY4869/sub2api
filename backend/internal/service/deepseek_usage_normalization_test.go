package service

import "testing"

func TestNormalizeOpenAIUsageForDisplayAndBilling_DeepSeek(t *testing.T) {
	result := normalizeOpenAIUsageForDisplayAndBilling(PlatformDeepSeek, OpenAIUsage{
		InputTokens:              1000,
		OutputTokens:             120,
		CacheCreationInputTokens: 200,
		CacheReadInputTokens:     300,
	})

	if result.DisplayTokens.InputTokens != 500 {
		t.Fatalf("DisplayTokens.InputTokens = %d, want 500", result.DisplayTokens.InputTokens)
	}
	if result.DisplayTokens.CacheCreationTokens != 200 {
		t.Fatalf("DisplayTokens.CacheCreationTokens = %d, want 200", result.DisplayTokens.CacheCreationTokens)
	}
	if result.DisplayTokens.CacheReadTokens != 300 {
		t.Fatalf("DisplayTokens.CacheReadTokens = %d, want 300", result.DisplayTokens.CacheReadTokens)
	}
	if result.BillingTokens.InputTokens != 700 {
		t.Fatalf("BillingTokens.InputTokens = %d, want 700", result.BillingTokens.InputTokens)
	}
	if result.BillingTokens.CacheCreationTokens != 0 {
		t.Fatalf("BillingTokens.CacheCreationTokens = %d, want 0", result.BillingTokens.CacheCreationTokens)
	}
	if result.BillingTokens.CacheReadTokens != 300 {
		t.Fatalf("BillingTokens.CacheReadTokens = %d, want 300", result.BillingTokens.CacheReadTokens)
	}
}

func TestNormalizeClaudeUsageForDisplayAndBilling_DeepSeek(t *testing.T) {
	result := normalizeClaudeUsageForDisplayAndBilling(PlatformDeepSeek, ClaudeUsage{
		InputTokens:              900,
		OutputTokens:             80,
		CacheCreationInputTokens: 150,
		CacheReadInputTokens:     250,
		CacheCreation5mTokens:    90,
		CacheCreation1hTokens:    60,
	})

	if result.DisplayTokens.InputTokens != 500 {
		t.Fatalf("DisplayTokens.InputTokens = %d, want 500", result.DisplayTokens.InputTokens)
	}
	if result.DisplayTokens.CacheCreationTokens != 150 {
		t.Fatalf("DisplayTokens.CacheCreationTokens = %d, want 150", result.DisplayTokens.CacheCreationTokens)
	}
	if result.DisplayTokens.CacheReadTokens != 250 {
		t.Fatalf("DisplayTokens.CacheReadTokens = %d, want 250", result.DisplayTokens.CacheReadTokens)
	}
	if result.DisplayTokens.CacheCreation5mTokens != 0 {
		t.Fatalf("DisplayTokens.CacheCreation5mTokens = %d, want 0", result.DisplayTokens.CacheCreation5mTokens)
	}
	if result.DisplayTokens.CacheCreation1hTokens != 0 {
		t.Fatalf("DisplayTokens.CacheCreation1hTokens = %d, want 0", result.DisplayTokens.CacheCreation1hTokens)
	}
	if result.BillingTokens.InputTokens != 650 {
		t.Fatalf("BillingTokens.InputTokens = %d, want 650", result.BillingTokens.InputTokens)
	}
	if result.BillingTokens.CacheCreationTokens != 0 {
		t.Fatalf("BillingTokens.CacheCreationTokens = %d, want 0", result.BillingTokens.CacheCreationTokens)
	}
	if result.BillingTokens.CacheReadTokens != 250 {
		t.Fatalf("BillingTokens.CacheReadTokens = %d, want 250", result.BillingTokens.CacheReadTokens)
	}
}

func TestNormalizeOpenAIUsageForDisplayAndBilling_NonDeepSeekUnchanged(t *testing.T) {
	result := normalizeOpenAIUsageForDisplayAndBilling(PlatformOpenAI, OpenAIUsage{
		InputTokens:              1000,
		OutputTokens:             120,
		CacheCreationInputTokens: 200,
		CacheReadInputTokens:     300,
	})

	if result.DisplayTokens.InputTokens != 700 {
		t.Fatalf("DisplayTokens.InputTokens = %d, want 700", result.DisplayTokens.InputTokens)
	}
	if result.DisplayTokens.CacheCreationTokens != 200 {
		t.Fatalf("DisplayTokens.CacheCreationTokens = %d, want 200", result.DisplayTokens.CacheCreationTokens)
	}
	if result.DisplayTokens.CacheReadTokens != 300 {
		t.Fatalf("DisplayTokens.CacheReadTokens = %d, want 300", result.DisplayTokens.CacheReadTokens)
	}
	if result.BillingTokens.InputTokens != 700 {
		t.Fatalf("BillingTokens.InputTokens = %d, want 700", result.BillingTokens.InputTokens)
	}
	if result.BillingTokens.CacheCreationTokens != 200 {
		t.Fatalf("BillingTokens.CacheCreationTokens = %d, want 200", result.BillingTokens.CacheCreationTokens)
	}
	if result.BillingTokens.CacheReadTokens != 300 {
		t.Fatalf("BillingTokens.CacheReadTokens = %d, want 300", result.BillingTokens.CacheReadTokens)
	}
}
