package service

import (
	"fmt"
	"sort"
	"strings"
)

func billingPricingCapabilitiesForRecord(record *modelCatalogRecord) BillingPricingCapabilities {
	provider := strings.TrimSpace(strings.ToLower(record.provider))
	supportsBatch := provider == PlatformGemini || provider == PlatformOpenAI || provider == PlatformAnthropic
	return BillingPricingCapabilities{
		SupportsTieredPricing:   record.longContextInputTokenThreshold > 0 || pricingHasThresholds(record.officialPricing) || pricingHasThresholds(record.salePricing) || pricingHasThresholds(record.upstreamPricing),
		SupportsBatchPricing:    supportsBatch,
		SupportsServiceTier:     record.supportsServiceTier,
		SupportsPromptCaching:   record.supportsPromptCaching,
		SupportsProviderSpecial: supportsBatch || record.supportsPromptCaching || provider == PlatformGemini,
	}
}

func pricingHasThresholds(pricing *ModelCatalogPricing) bool {
	return pricing != nil && ((pricing.InputTokenThreshold != nil && pricing.InputCostPerTokenAboveThreshold != nil) ||
		(pricing.OutputTokenThreshold != nil && pricing.OutputCostPerTokenAboveThreshold != nil))
}

func pricingItemsForRecord(record *modelCatalogRecord, layer string, rules []BillingRule) []BillingPriceItem {
	if record == nil {
		return []BillingPriceItem{}
	}
	items := make([]BillingPriceItem, 0, 24)
	pricing := selectGeminiMatrixPricing(record, layer)
	if !isGeminiBillingCompatModel(record.model) {
		items = append(items, pricingItemsFromFlatPricing(record, layer, pricing)...)
	}
	items = append(items, pricingItemsFromRules(record, layer, rules)...)
	if isGeminiBillingCompatModel(record.model) {
		items = append(items, pricingItemsFromGeminiMatrix(record, layer, rules)...)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ChargeSlot == items[j].ChargeSlot {
			return items[i].ID < items[j].ID
		}
		return items[i].ChargeSlot < items[j].ChargeSlot
	})
	return dedupeBillingPriceItems(items)
}

func pricingItemsFromFlatPricing(record *modelCatalogRecord, layer string, pricing *ModelCatalogPricing) []BillingPriceItem {
	if record == nil || pricing == nil {
		return []BillingPriceItem{}
	}
	items := make([]BillingPriceItem, 0, 16)
	appendBase := func(chargeSlot, unit string, base *float64, priority *float64, threshold *int, above *float64, priorityAbove *float64) {
		if base != nil {
			item := BillingPriceItem{
				ID:         billingBaseItemID(layer, chargeSlot, ""),
				ChargeSlot: chargeSlot,
				Unit:       unit,
				Layer:      normalizeBillingDimension(layer, BillingLayerSale),
				Mode:       BillingPriceItemModeBase,
				Price:      *base,
				Enabled:    true,
			}
			if threshold != nil && above != nil {
				item.Mode = BillingPriceItemModeTiered
				item.ThresholdTokens = modelCatalogIntPtr(*threshold)
				item.PriceAboveThresh = modelCatalogFloat64Ptr(*above)
			}
			items = append(items, item)
		}
		if priority != nil {
			item := BillingPriceItem{
				ID:          billingBaseItemID(layer, chargeSlot, BillingServiceTierPriority),
				ChargeSlot:  chargeSlot,
				Unit:        unit,
				Layer:       normalizeBillingDimension(layer, BillingLayerSale),
				Mode:        BillingPriceItemModeServiceTier,
				ServiceTier: BillingServiceTierPriority,
				Price:       *priority,
				Enabled:     true,
			}
			if threshold != nil && priorityAbove != nil {
				item.Mode = BillingPriceItemModeTiered
				item.ThresholdTokens = modelCatalogIntPtr(*threshold)
				item.PriceAboveThresh = modelCatalogFloat64Ptr(*priorityAbove)
			}
			items = append(items, item)
		}
	}

	appendBase(BillingChargeSlotTextInput, BillingUnitInputToken, pricing.InputCostPerToken, pricing.InputCostPerTokenPriority, pricing.InputTokenThreshold, pricing.InputCostPerTokenAboveThreshold, pricing.InputCostPerTokenPriorityAboveThreshold)
	appendBase(BillingChargeSlotTextOutput, BillingUnitOutputToken, pricing.OutputCostPerToken, pricing.OutputCostPerTokenPriority, pricing.OutputTokenThreshold, pricing.OutputCostPerTokenAboveThreshold, pricing.OutputCostPerTokenPriorityAboveThreshold)
	appendBase(BillingChargeSlotCacheCreate, BillingUnitCacheCreateToken, pricing.CacheCreationInputTokenCost, nil, nil, nil, nil)
	appendBase(BillingChargeSlotCacheRead, BillingUnitCacheReadToken, pricing.CacheReadInputTokenCost, pricing.CacheReadInputTokenCostPriority, nil, nil, nil)
	appendBase(BillingChargeSlotImageOutput, BillingUnitImage, pricing.OutputCostPerImage, pricing.OutputCostPerImagePriority, nil, nil, nil)
	appendBase(BillingChargeSlotVideoRequest, BillingUnitVideoRequest, pricing.OutputCostPerVideoRequest, nil, nil, nil, nil)

	if pricing.CacheCreationInputTokenCostAbove1hr != nil {
		items = append(items, BillingPriceItem{
			ID:          billingBaseItemID(layer, BillingChargeSlotCacheStorageTokenHour, ""),
			ChargeSlot:  BillingChargeSlotCacheStorageTokenHour,
			Unit:        BillingUnitCacheStorageTokenHour,
			Layer:       normalizeBillingDimension(layer, BillingLayerSale),
			Mode:        BillingPriceItemModeProviderRule,
			OperationType: "cache_storage",
			Price:       *pricing.CacheCreationInputTokenCostAbove1hr,
			Enabled:     true,
		})
	}

	items = append(items, defaultBatchFormulaItems(record, layer, items)...)
	return items
}

func defaultBatchFormulaItems(record *modelCatalogRecord, layer string, source []BillingPriceItem) []BillingPriceItem {
	if record == nil || !billingPricingCapabilitiesForRecord(record).SupportsBatchPricing {
		return []BillingPriceItem{}
	}
	items := make([]BillingPriceItem, 0, len(source))
	for _, item := range source {
		if item.Mode == BillingPriceItemModeBatch || item.Price <= 0 || item.ChargeSlot == BillingChargeSlotCacheStorageTokenHour {
			continue
		}
		multiplier := 0.5
		items = append(items, BillingPriceItem{
			ID:               billingBaseItemID(layer, item.ChargeSlot, "batch"),
			ChargeSlot:       item.ChargeSlot,
			Unit:             item.Unit,
			Layer:            normalizeBillingDimension(layer, BillingLayerSale),
			Mode:             BillingPriceItemModeBatch,
			BatchMode:        BillingBatchModeBatch,
			OperationType:    operationTypeForChargeSlot(item.ChargeSlot),
			Price:            item.Price * multiplier,
			FormulaSource:    item.ID,
			FormulaMultiplier: modelCatalogFloat64Ptr(multiplier),
			Enabled:          true,
		})
	}
	return items
}

func pricingItemsFromRules(record *modelCatalogRecord, layer string, rules []BillingRule) []BillingPriceItem {
	if record == nil {
		return []BillingPriceItem{}
	}
	items := make([]BillingPriceItem, 0, 8)
	for _, rule := range rules {
		rule = normalizeBillingRule(rule)
		if !rule.Enabled || !billingRuleMatchesModel(rule, record.model) || rule.Layer != normalizeBillingDimension(layer, BillingLayerSale) {
			continue
		}
		if isGeminiMatrixRule(rule) || isGeminiCompatRule(rule, layer) {
			continue
		}
		slot, ok := geminiMatrixSlotForRule(rule)
		if !ok {
			continue
		}
		mode := BillingPriceItemModeProviderRule
		if rule.BatchMode == BillingBatchModeBatch {
			mode = BillingPriceItemModeBatch
		} else if billingRuleUsesExplicitValue(rule.ServiceTier) && rule.ServiceTier != BillingServiceTierStandard {
			mode = BillingPriceItemModeServiceTier
		}
		items = append(items, BillingPriceItem{
			ID:            rule.ID,
			ChargeSlot:    slot,
			Unit:          rule.Unit,
			Layer:         rule.Layer,
			Mode:          mode,
			ServiceTier:   rule.ServiceTier,
			BatchMode:     rule.BatchMode,
			Surface:       rule.Surface,
			OperationType: rule.OperationType,
			InputModality: rule.Matchers.InputModality,
			OutputModality: rule.Matchers.OutputModality,
			CachePhase:    rule.Matchers.CachePhase,
			GroundingKind: rule.Matchers.GroundingKind,
			ContextWindow: rule.Matchers.ContextWindow,
			Price:         rule.Price,
			RuleID:        rule.ID,
			Enabled:       rule.Enabled,
		})
	}
	return items
}

func pricingItemsFromGeminiMatrix(record *modelCatalogRecord, layer string, rules []BillingRule) []BillingPriceItem {
	matrix := buildGeminiMatrixForRecord(record, layer, rules)
	items := make([]BillingPriceItem, 0, len(matrix.Rows)*4)
	for _, row := range matrix.Rows {
		for slot, cell := range row.Slots {
			if cell.Price == nil {
				continue
			}
			spec, ok := geminiMatrixSlotSpecs[slot]
			if !ok {
				continue
			}
			mode := BillingPriceItemModeProviderRule
			if row.ServiceTier != "" && row.ServiceTier != BillingServiceTierStandard {
				mode = BillingPriceItemModeServiceTier
			}
			items = append(items, BillingPriceItem{
				ID:             geminiMatrixRuleID(record.model, layer, row.Surface, row.ServiceTier, slot),
				ChargeSlot:     slot,
				Unit:           spec.unit,
				Layer:          normalizeBillingDimension(layer, BillingLayerSale),
				Mode:           mode,
				ServiceTier:    row.ServiceTier,
				Surface:        row.Surface,
				OperationType:  spec.operation,
				InputModality:  spec.matchers.InputModality,
				OutputModality: spec.matchers.OutputModality,
				CachePhase:     spec.matchers.CachePhase,
				GroundingKind:  spec.matchers.GroundingKind,
				ContextWindow:  spec.matchers.ContextWindow,
				Price:          *cell.Price,
				RuleID:         cell.RuleID,
				DerivedVia:     cell.DerivedVia,
				Enabled:        true,
			})
		}
	}
	return items
}

func dedupeBillingPriceItems(items []BillingPriceItem) []BillingPriceItem {
	seen := make(map[string]struct{}, len(items))
	filtered := make([]BillingPriceItem, 0, len(items))
	for _, item := range items {
		key := billingPriceItemKey(item)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		filtered = append(filtered, normalizeBillingPriceItem(item))
	}
	return filtered
}

func billingPriceItemKey(item BillingPriceItem) string {
	return strings.Join([]string{
		normalizeBillingDimension(item.Layer, BillingLayerSale),
		normalizeBillingDimension(item.ChargeSlot, ""),
		normalizeBillingDimension(item.Unit, ""),
		normalizeBillingDimension(item.ServiceTier, ""),
		normalizeBillingActualBatchMode(item.BatchMode),
		normalizeBillingSurface(item.Surface),
		normalizeBillingDimension(item.OperationType, ""),
		normalizeBillingDimension(item.InputModality, ""),
		normalizeBillingDimension(item.OutputModality, ""),
		normalizeBillingDimension(item.CachePhase, ""),
		normalizeBillingDimension(item.GroundingKind, ""),
		normalizeBillingDimension(item.ContextWindow, ""),
	}, "|")
}

func normalizeBillingPriceItem(item BillingPriceItem) BillingPriceItem {
	item.Layer = normalizeBillingDimension(item.Layer, BillingLayerSale)
	item.ChargeSlot = normalizeBillingDimension(item.ChargeSlot, "")
	item.Unit = normalizeBillingDimension(item.Unit, "")
	item.ServiceTier = normalizeBillingServiceTier(item.ServiceTier)
	item.BatchMode = normalizeBillingBatchMode(item.BatchMode)
	item.Surface = normalizeBillingSurface(item.Surface)
	item.OperationType = normalizeBillingDimension(item.OperationType, operationTypeForChargeSlot(item.ChargeSlot))
	item.InputModality = normalizeBillingDimension(item.InputModality, "")
	item.OutputModality = normalizeBillingDimension(item.OutputModality, "")
	item.CachePhase = normalizeBillingDimension(item.CachePhase, "")
	item.GroundingKind = normalizeBillingDimension(item.GroundingKind, "")
	item.ContextWindow = normalizeBillingDimension(item.ContextWindow, "")
	if item.ID == "" {
		item.ID = billingGeneratedRuleID(item)
	}
	item.Enabled = item.Enabled || item.Price > 0
	return item
}

func billingBaseItemID(layer, chargeSlot, suffix string) string {
	key := []string{"sheet", normalizeBillingDimension(layer, BillingLayerSale), normalizeBillingDimension(chargeSlot, "")}
	if trimmed := strings.TrimSpace(suffix); trimmed != "" {
		key = append(key, trimmed)
	}
	return strings.Join(key, "__")
}

func billingGeneratedRuleID(item BillingPriceItem) string {
	return fmt.Sprintf(
		"%s__%s__%s__%s__%s__%s",
		billingPricingRuleIDPrefix,
		normalizeBillingDimension(item.Layer, BillingLayerSale),
		normalizeBillingDimension(item.ChargeSlot, ""),
		normalizeBillingDimension(item.ServiceTier, "default"),
		normalizeBillingActualBatchMode(item.BatchMode),
		normalizeBillingSurface(item.Surface),
	)
}

func operationTypeForChargeSlot(slot string) string {
	switch normalizeBillingDimension(slot, "") {
	case BillingChargeSlotCacheCreate, BillingChargeSlotCacheRead:
		return "cache_usage"
	case BillingChargeSlotCacheStorageTokenHour:
		return "cache_storage"
	case BillingChargeSlotFileSearchEmbeddingToken:
		return "file_search_embedding"
	case BillingChargeSlotFileSearchRetrievalToken:
		return "file_search_retrieval"
	default:
		return "generate_content"
	}
}
