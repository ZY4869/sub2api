package service

import "context"

func applyBillingRuntimeResultMetadataToContext(ctx context.Context, result *BillingRuntimeResult) {
	if result == nil {
		return
	}
	if result.Classification != nil {
		applyGeminiBillingMetadataToContext(ctx, &GeminiBillingCalculationResult{
			Classification: result.Classification,
			MatchedRuleIDs: result.MatchedItems,
			Fallback:       &BillingSimulationFallback{Reason: result.FallbackReason},
		})
		return
	}
	if len(result.MatchedItems) > 0 {
		SetBillingRuleIDMetadata(ctx, result.MatchedItems[0])
	}
}
