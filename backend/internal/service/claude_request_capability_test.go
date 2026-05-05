package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/claude"
	"github.com/stretchr/testify/require"
)

func TestParseClaudeRequestCapability_StripsMillionContextSuffixAndNormalizesEffort(t *testing.T) {
	capability := ParseClaudeRequestCapability("  claude-sonnet-4.5[1m]  ", " MAX ")

	require.Equal(t, "claude-sonnet-4.5[1m]", capability.RequestedModelRaw)
	require.Equal(t, "claude-sonnet-4.5", capability.RequestedModelNormalized)
	require.Equal(t, "max", capability.GatewayEffortLevel)
	require.True(t, capability.MillionContextRequested)
	require.True(t, capability.MillionContextSupported)
	require.Equal(t, "model_suffix_[1m]", capability.MillionContextSource)
	require.Equal(t, claude.BetaContext1M, capability.MillionContextBetaToken)
}

func TestParseClaudeRequestCapability_DeepSeekMillionContextSupport(t *testing.T) {
	flashCapability := ParseClaudeRequestCapability("deepseek-v4-flash[1m]", "")
	require.True(t, flashCapability.MillionContextRequested)
	require.True(t, flashCapability.MillionContextSupported)
	require.Equal(t, "deepseek-v4-flash", flashCapability.RequestedModelNormalized)

	proCapability := ParseClaudeRequestCapability("deepseek-v4-pro[1m]", "")
	require.True(t, proCapability.MillionContextRequested)
	require.True(t, proCapability.MillionContextSupported)
	require.Equal(t, "deepseek-v4-pro", proCapability.RequestedModelNormalized)
}

func TestResolveClaudeRequestCapabilityForRuntime_AppliesOnlyToAnthropicFamily(t *testing.T) {
	capability := ResolveClaudeRequestCapabilityForRuntime("claude-sonnet-4.5[1m]", "", PlatformAnthropic)
	require.True(t, capability.MillionContextEffective)

	openAICapability := ResolveClaudeRequestCapabilityForRuntime("claude-sonnet-4.5[1m]", "", PlatformOpenAI)
	require.True(t, openAICapability.MillionContextRequested)
	require.False(t, openAICapability.MillionContextEffective)
}

func TestResolveClaudeRequestCapabilityForRuntime_UnsupportedModelStaysRequestedOnly(t *testing.T) {
	capability := ResolveClaudeRequestCapabilityForRuntime("gpt-5[1m]", "", PlatformAnthropic)

	require.True(t, capability.MillionContextRequested)
	require.False(t, capability.MillionContextSupported)
	require.False(t, capability.MillionContextEffective)
	require.Equal(t, "gpt-5", capability.RequestedModelNormalized)
}

func TestRecordClaudeCapabilityMetadata_StoresRequestedAndEffectiveFlags(t *testing.T) {
	ctx := EnsureRequestMetadata(context.Background())
	capability := ClaudeRequestCapability{
		RequestedModelRaw:        "deepseek-v4-pro[1m]",
		RequestedModelNormalized: "deepseek-v4-pro",
		MillionContextRequested:  true,
		MillionContextEffective:  true,
		MillionContextSource:     "model_suffix_[1m]",
		MillionContextBetaToken:  claude.BetaContext1M,
	}

	RecordClaudeCapabilityMetadata(ctx, capability)

	raw, ok := ClaudeRequestedModelRawMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "deepseek-v4-pro[1m]", raw)

	normalized, ok := ClaudeRequestedModelNormalizedMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "deepseek-v4-pro", normalized)

	requested, ok := ClaudeMillionContextRequestedMetadataFromContext(ctx)
	require.True(t, ok)
	require.True(t, requested)

	effective, ok := ClaudeMillionContextEffectiveMetadataFromContext(ctx)
	require.True(t, ok)
	require.True(t, effective)

	source, ok := ClaudeMillionContextSourceMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, "model_suffix_[1m]", source)

	betaToken, ok := ClaudeMillionContextBetaTokenMetadataFromContext(ctx)
	require.True(t, ok)
	require.Equal(t, claude.BetaContext1M, betaToken)
}

func TestRecordClaudeCapabilityMetadata_ClearsBetaTokenWhenNotEffective(t *testing.T) {
	ctx := EnsureRequestMetadata(context.Background())
	RecordClaudeCapabilityMetadata(ctx, ClaudeRequestCapability{
		RequestedModelRaw:        "gpt-5[1m]",
		RequestedModelNormalized: "gpt-5",
		MillionContextRequested:  true,
		MillionContextEffective:  false,
		MillionContextSource:     "model_suffix_[1m]",
		MillionContextBetaToken:  claude.BetaContext1M,
	})

	betaToken, ok := ClaudeMillionContextBetaTokenMetadataFromContext(ctx)
	require.False(t, ok)
	require.Equal(t, "", betaToken)
}
