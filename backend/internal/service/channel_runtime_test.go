package service

import (
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/model"
	"github.com/stretchr/testify/require"
)

func TestChannelService_ResolveUsagePricing_UsesPlatformScopedPricing(t *testing.T) {
	anthropicPrice := 0.21
	antigravityPrice := 0.11
	channel := &model.Channel{
		ModelPricing: []model.ChannelModelPricing{
			{ID: 1, Platform: PlatformAnthropic, Models: []string{"claude-*"}, BillingMode: model.ChannelBillingModeToken, InputPrice: &anthropicPrice},
			{ID: 2, Platform: PlatformAntigravity, Models: []string{"claude-*"}, BillingMode: model.ChannelBillingModeToken, InputPrice: &antigravityPrice},
		},
	}
	state := &GatewayChannelState{Channel: channel, Platform: PlatformAntigravity}

	resolved := (&ChannelService{}).ResolveUsagePricing(state, "claude-3-7-sonnet", GatewayChannelUsage{TotalTokens: 128})
	require.NotNil(t, resolved)
	require.Equal(t, int64(2), resolved.PricingID)
	require.NotNil(t, resolved.InputPrice)
	require.Equal(t, antigravityPrice, *resolved.InputPrice)
}

func TestResolveChannelMappingTarget_UsesPlatformScopedMapping(t *testing.T) {
	channel := &model.Channel{
		ModelMapping: map[string]map[string]string{
			PlatformAnthropic:   {"claude-3-7-sonnet": "anthropic-target"},
			PlatformAntigravity: {"claude-3-7-sonnet": "ag-target"},
			"*":                 {"claude-3-7-opus": "fallback-target"},
		},
	}

	require.Equal(t, "ag-target", resolveChannelMappingTarget(channel, PlatformAntigravity, "claude-3-7-sonnet"))
	require.Equal(t, "fallback-target", resolveChannelMappingTarget(channel, PlatformAntigravity, "claude-3-7-opus"))
}
