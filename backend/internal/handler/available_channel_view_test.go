//go:build unit

package handler

import (
	"encoding/json"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/stretchr/testify/require"
)

func TestBuildPlatformSections_FiltersSupportedModelsByVisibleGroupPlatforms(t *testing.T) {
	ch := service.AvailableChannel{
		Name:        "Channel One",
		Description: "desc",
		Groups: []service.AvailableGroupRef{
			{ID: 10, Name: "OpenAI", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1, IsExclusive: false},
			{ID: 20, Name: "Anthropic", Platform: service.PlatformAnthropic, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1, IsExclusive: false},
		},
		SupportedModels: []service.SupportedModel{
			{Name: "gpt-4o", Platform: service.PlatformOpenAI, Pricing: &service.SupportedModelPricing{BillingMode: "token"}},
			{Name: "claude-3-5-sonnet-20241022", Platform: service.PlatformAnthropic, Pricing: &service.SupportedModelPricing{BillingMode: "token"}},
		},
	}

	allowed := map[int64]struct{}{10: {}}
	visibleGroups := filterUserVisibleGroups(ch.Groups, allowed)
	require.Len(t, visibleGroups, 1)
	require.Equal(t, int64(10), visibleGroups[0].ID)
	require.Equal(t, service.PlatformOpenAI, visibleGroups[0].Platform)

	sections := buildPlatformSections(ch, visibleGroups)
	require.Len(t, sections, 1)
	require.Equal(t, service.PlatformOpenAI, sections[0].Platform)
	require.Len(t, sections[0].Groups, 1)
	require.Len(t, sections[0].SupportedModels, 1)
	require.Equal(t, "gpt-4o", sections[0].SupportedModels[0].Name)
	require.Equal(t, service.PlatformOpenAI, sections[0].SupportedModels[0].Platform)
}

func TestUserAvailableChannel_DoesNotExposeInternalFields(t *testing.T) {
	payload := userAvailableChannel{
		Name:        "Channel One",
		Description: "desc",
		Platforms: []userChannelPlatformSection{
			{
				Platform: service.PlatformOpenAI,
				Groups: []userAvailableGroup{
					{ID: 10, Name: "OpenAI", Platform: service.PlatformOpenAI, SubscriptionType: service.SubscriptionTypeStandard, RateMultiplier: 1, IsExclusive: false},
				},
				SupportedModels: []userSupportedModel{
					{Name: "gpt-4o", Platform: service.PlatformOpenAI, Pricing: nil},
				},
			},
		},
	}

	raw, err := json.Marshal(payload)
	require.NoError(t, err)
	var decoded map[string]any
	require.NoError(t, json.Unmarshal(raw, &decoded))
	_, hasID := decoded["id"]
	_, hasStatus := decoded["status"]
	require.False(t, hasID)
	require.False(t, hasStatus)
}

func TestToUserSupportedModelPricing_MapsIntervals(t *testing.T) {
	pricing := &service.SupportedModelPricing{
		BillingMode: "token",
		Intervals: []service.SupportedModelPricingInterval{
			{
				MinTokens: 10,
				MaxTokens: ptrInt(100),
				TierLabel: "tier-1",
			},
		},
	}

	out := toUserSupportedModelPricing(pricing)
	require.NotNil(t, out)
	require.Equal(t, "token", out.BillingMode)
	require.Len(t, out.Intervals, 1)
	require.Equal(t, 10, out.Intervals[0].MinTokens)
	require.Equal(t, 100, *out.Intervals[0].MaxTokens)
	require.Equal(t, "tier-1", out.Intervals[0].TierLabel)
}

func ptrInt(v int) *int {
	return &v
}
