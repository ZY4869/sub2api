package service

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestModelPlazaBuildGroupsVisibilityFilteringAndDTOPrivacy(t *testing.T) {
	imagePrice := 0.02
	webSearchPrice := 0.03
	tokenPrice := 0.000001
	channels := []AvailableChannel{
		{
			Status: StatusActive,
			Groups: []AvailableGroupRef{
				{ID: 2, Name: "Exclusive Anthropic", Platform: PlatformAnthropic, IsExclusive: true, RateMultiplier: 1.5},
				{ID: 1, Name: "Public OpenAI", Platform: PlatformOpenAI, RateMultiplier: 1, ImagePrice1K: &imagePrice, WebSearchPricePerCall: &webSearchPrice},
			},
			SupportedModels: []SupportedModel{
				{Name: "gpt-4o", Platform: PlatformOpenAI, Pricing: &SupportedModelPricing{BillingMode: "token", InputPrice: &tokenPrice}},
				{Name: "claude-3-5-sonnet-20241022", Platform: PlatformAnthropic},
				{Name: "gpt-4o", Platform: PlatformOpenAI},
			},
		},
		{
			Status: StatusDisabled,
			Groups: []AvailableGroupRef{
				{ID: 3, Name: "Inactive", Platform: PlatformOpenAI},
			},
			SupportedModels: []SupportedModel{{Name: "should-not-appear", Platform: PlatformOpenAI}},
		},
	}

	anonymous := BuildModelPlazaGroups(channels, nil, false)
	require.Len(t, anonymous, 1)
	require.Equal(t, int64(1), anonymous[0].ID)
	require.True(t, anonymous[0].ImageRateIndependent)
	require.Equal(t, &imagePrice, anonymous[0].ImagePrice1K)
	require.Equal(t, &webSearchPrice, anonymous[0].WebSearchPricePerCall)
	require.Len(t, anonymous[0].Models, 1)
	require.Equal(t, "gpt-4o", anonymous[0].Models[0].Name)
	require.Equal(t, PlatformOpenAI, anonymous[0].Models[0].Platform)

	authenticated := BuildModelPlazaGroups(channels, map[int64]struct{}{2: {}}, true)
	require.Len(t, authenticated, 2)
	require.Equal(t, int64(2), authenticated[0].ID)
	require.Equal(t, "claude-3-5-sonnet-20241022", authenticated[0].Models[0].Name)
	require.Equal(t, int64(1), authenticated[1].ID)

	body, err := json.Marshal(anonymous)
	require.NoError(t, err)
	require.Contains(t, string(body), "gpt-4o")
	require.NotContains(t, string(body), "target_model_id")
	require.NotContains(t, string(body), "should-not-appear")
}
