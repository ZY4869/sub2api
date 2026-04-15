package service

import "testing"

func TestExtractGeminiUsage_InteractionsOfficialTotals(t *testing.T) {
	t.Parallel()

	usage := extractGeminiUsage([]byte(`{"usage":{"total_input_tokens":120,"total_output_tokens":45,"total_cached_tokens":20,"total_reasoning_tokens":7}}`))
	if usage == nil {
		t.Fatalf("expected usage to be parsed")
	}
	if usage.InputTokens != 100 || usage.OutputTokens != 52 || usage.CacheReadInputTokens != 20 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestExtractGeminiUsage_InteractionsThoughtTotalsFallback(t *testing.T) {
	t.Parallel()

	usage := extractGeminiUsage([]byte(`{"usage":{"total_input_tokens":18,"total_output_tokens":9,"total_cached_tokens":4,"total_thought_tokens":3}}`))
	if usage == nil {
		t.Fatalf("expected usage to be parsed")
	}
	if usage.InputTokens != 14 || usage.OutputTokens != 12 || usage.CacheReadInputTokens != 4 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}

func TestExtractGeminiUsage_InteractionsUnknownUsageFieldsAreIgnored(t *testing.T) {
	t.Parallel()

	usage := extractGeminiUsage([]byte(`{"usage":{"total_input_tokens":24,"total_output_tokens":8,"total_cached_tokens":6,"total_reasoning_tokens":2,"future_usage_field":99}}`))
	if usage == nil {
		t.Fatalf("expected usage to be parsed")
	}
	if usage.InputTokens != 18 || usage.OutputTokens != 10 || usage.CacheReadInputTokens != 6 {
		t.Fatalf("unexpected usage: %+v", usage)
	}
}
