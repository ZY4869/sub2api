package service

import (
	"context"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/ctxkey"
	"github.com/stretchr/testify/require"
)

func TestApplyGroupOpenAIReasoningPolicy_MapsAndClampsEffectiveEffort(t *testing.T) {
	raw := "max"
	effective := "xhigh"
	group := &Group{
		ID:                 11,
		Platform:           PlatformOpenAI,
		Status:             StatusActive,
		Hydrated:           true,
		MaxReasoningEffort: "medium",
		ReasoningEffortMappings: []ReasoningEffortMapping{
			{Model: "gpt-5.4-*", From: "x-high", To: "high"},
		},
	}

	got := ApplyGroupOpenAIReasoningPolicy(group, GatewayEffortResolution{
		Raw:       &raw,
		Effective: &effective,
		Source:    effortSourceOpenAIField,
	}, "gpt-5.4-pro")

	require.NotNil(t, got.Raw)
	require.Equal(t, "max", *got.Raw)
	require.NotNil(t, got.Effective)
	require.Equal(t, "medium", *got.Effective)
	require.Equal(t, effortSourceOpenAIField, got.Source)
}

func TestApplyGroupOpenAIReasoningPolicy_AppliesContextGroupForWebSocketAndHTTPCallers(t *testing.T) {
	raw := "high"
	group := &Group{
		ID:                 12,
		Platform:           PlatformOpenAI,
		Status:             StatusActive,
		Hydrated:           true,
		MaxReasoningEffort: "low",
	}
	ctx := context.WithValue(context.Background(), ctxkey.Group, group)

	got := ApplyContextOpenAIReasoningPolicy(ctx, GatewayEffortResolution{
		Raw:       &raw,
		Effective: &raw,
	}, "gpt-5.4")

	require.NotNil(t, got.Raw)
	require.Equal(t, "high", *got.Raw)
	require.NotNil(t, got.Effective)
	require.Equal(t, "low", *got.Effective)
}

func TestNormalizeReasoningEffortMappings_SupportsAliasesAndDropsInvalidRows(t *testing.T) {
	got := NormalizeReasoningEffortMappings([]ReasoningEffortMapping{
		{Model: " gpt-* ", From: "x-high", ReasoningEffort: "max"},
		{Model: "", To: "low"},
		{Model: "gpt-5", To: "turbo"},
	})

	require.Equal(t, []ReasoningEffortMapping{
		{Model: "gpt-*", From: "xhigh", To: "max", ReasoningEffort: "max"},
	}, got)
}
