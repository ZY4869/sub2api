package service

import (
	"bytes"
	"strings"
	"testing"
	"time"
)

type testGeminiStreamFlusher struct{}

func (testGeminiStreamFlusher) Flush() {}

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

func TestGeminiClaudeStreamEmitter_ClosesToolBlockBeforeText(t *testing.T) {
	t.Parallel()

	var out bytes.Buffer
	emitter := newGeminiClaudeStreamEmitter(&out, testGeminiStreamFlusher{}, time.Now(), "gemini-test")
	emitter.consumeResponse(map[string]any{
		"candidates": []any{
			map[string]any{
				"content": map[string]any{
					"parts": []any{
						map[string]any{
							"functionCall": map[string]any{
								"name": "lookup",
								"args": map[string]any{"q": "status"},
							},
						},
						map[string]any{"text": "done"},
					},
				},
			},
		},
	}, nil)

	body := out.String()
	toolStart := strings.Index(body, `"type":"tool_use"`)
	toolStop := -1
	if toolStart >= 0 {
		if offset := strings.Index(body[toolStart:], `"type":"content_block_stop"`); offset >= 0 {
			toolStop = toolStart + offset
		}
	}
	textStart := strings.Index(body, `"content_block":{"text":"","type":"text"}`)
	textDelta := strings.Index(body, `"text":"done"`)

	if toolStart < 0 || toolStop < 0 || textStart < 0 || textDelta < 0 {
		t.Fatalf("expected tool start/stop and text events, got:\n%s", body)
	}
	if toolStart >= toolStop || toolStop >= textStart || textStart >= textDelta {
		t.Fatalf("expected tool block to close before text block starts, got:\n%s", body)
	}
}
