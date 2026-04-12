package service

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestMatchBillingRule_PrefersSurfaceThenOperationThenModelSpecificity(t *testing.T) {
	rules := []BillingRule{
		{
			ID:            "any",
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Surface:       BillingSurfaceAny,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			Unit:          BillingUnitInputToken,
			Price:         1,
			Enabled:       true,
		},
		{
			ID:            "native",
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			Unit:          BillingUnitInputToken,
			Price:         2,
			Enabled:       true,
		},
		{
			ID:            "native-model",
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			Matchers:      BillingRuleMatchers{Models: []string{"gemini-2.5-pro"}},
			Unit:          BillingUnitInputToken,
			Price:         3,
			Enabled:       true,
		},
	}

	selected := matchBillingRule(rules, billingUnitDemand{
		chargeSlot: BillingChargeSlotTextInput,
		unit:       BillingUnitInputToken,
		count:      128,
		context: billingMatchContext{
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Model:         "gemini-2.5-pro",
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			InputModality: "text",
			ContextWindow: BillingContextWindowStandard,
		},
	})

	require.NotNil(t, selected)
	require.Equal(t, "native-model", selected.ID)
}

func TestMatchBillingRule_PrefersPriorityThenStableIDTieBreak(t *testing.T) {
	rules := []BillingRule{
		{
			ID:            "rule-b",
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			Unit:          BillingUnitInputToken,
			Priority:      20,
			Price:         2,
			Enabled:       true,
		},
		{
			ID:            "rule-a",
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			Unit:          BillingUnitInputToken,
			Priority:      10,
			Price:         1,
			Enabled:       true,
		},
	}

	selected := matchBillingRule(rules, billingUnitDemand{
		chargeSlot: BillingChargeSlotTextInput,
		unit:       BillingUnitInputToken,
		count:      64,
		context: billingMatchContext{
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Model:         "gemini-2.5-flash",
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			InputModality: "text",
			ContextWindow: BillingContextWindowStandard,
		},
	})

	require.NotNil(t, selected)
	require.Equal(t, "rule-a", selected.ID)

	rules[0].Priority = 10
	selected = matchBillingRule(rules, billingUnitDemand{
		chargeSlot: BillingChargeSlotTextInput,
		unit:       BillingUnitInputToken,
		count:      64,
		context: billingMatchContext{
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Model:         "gemini-2.5-flash",
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			InputModality: "text",
			ContextWindow: BillingContextWindowStandard,
		},
	})
	require.NotNil(t, selected)
	require.Equal(t, "rule-a", selected.ID)
}

func TestDescribeUnmatchedDemand_ReturnsSpecificMissReason(t *testing.T) {
	rules := []BillingRule{
		{
			ID:            "sale-native",
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Surface:       BillingSurfaceGeminiNative,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			Unit:          BillingUnitInputToken,
			Enabled:       true,
		},
	}

	demand := billingUnitDemand{
		chargeSlot: BillingChargeSlotTextInput,
		unit:       BillingUnitInputToken,
		count:      32,
		context: billingMatchContext{
			Provider:      BillingRuleProviderGemini,
			Layer:         BillingLayerSale,
			Model:         "gemini-2.5-pro",
			Surface:       BillingSurfaceOpenAICompat,
			OperationType: "generate_content",
			ServiceTier:   BillingServiceTierStandard,
			BatchMode:     BillingBatchModeRealtime,
			InputModality: "text",
			ContextWindow: BillingContextWindowStandard,
		},
	}

	unmatched := describeUnmatchedDemand(rules, demand)
	require.Equal(t, "surface_miss", unmatched.Reason)
	require.Equal(t, []string{"surface"}, unmatched.MissingDimensions)
}
