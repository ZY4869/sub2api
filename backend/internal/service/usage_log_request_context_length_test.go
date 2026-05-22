package service

import "testing"

import "github.com/stretchr/testify/require"

func TestResolveUsageLogRequestContextLengthTokens_ExplicitMillionContext(t *testing.T) {
	raw := "claude-opus-4-7[1m]"
	log := &UsageLog{
		Model:                   "claude-opus-4-7",
		RequestedModel:          "claude-opus-4-7",
		RequestedModelRaw:       &raw,
		MillionContextRequested: requestContextBoolPtr(true),
	}

	got := ResolveUsageLogRequestContextLengthTokens(log)
	require.NotNil(t, got)
	require.Equal(t, 1_000_000, *got)
}

func TestResolveUsageLogRequestContextLengthTokens_DefaultRegistryContext(t *testing.T) {
	log := &UsageLog{
		Model:          "claude-opus-4-7",
		RequestedModel: "claude-opus-4-7",
	}

	got := ResolveUsageLogRequestContextLengthTokens(log)
	require.NotNil(t, got)
	require.Equal(t, 200_000, *got)
}

func TestResolveUsageLogRequestContextLengthTokens_UnresolvedReturnsNil(t *testing.T) {
	log := &UsageLog{
		Model:          "unknown-model-x",
		RequestedModel: "unknown-model-x",
	}

	got := ResolveUsageLogRequestContextLengthTokens(log)
	require.Nil(t, got)
}

func requestContextBoolPtr(value bool) *bool {
	return &value
}
