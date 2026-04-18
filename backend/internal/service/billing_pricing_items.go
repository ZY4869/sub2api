package service

import (
	"fmt"
	"math"
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
	if isGeminiBillingCompatModel(record.model) {
		items := pricingItemsForGeminiRecord(record, layer, rules)
		sort.SliceStable(items, func(i, j int) bool {
			if items[i].ChargeSlot == items[j].ChargeSlot {
				return items[i].ID < items[j].ID
			}
			return items[i].ChargeSlot < items[j].ChargeSlot
		})
		return prepareBillingPriceItemsForEditor(items)
	}
	items := make([]BillingPriceItem, 0, 24)
	pricing := selectGeminiMatrixPricing(record, layer)
	items = append(items, pricingItemsFromFlatPricing(record, layer, pricing)...)
	items = append(items, pricingItemsFromRules(record, layer, rules)...)
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].ChargeSlot == items[j].ChargeSlot {
			return items[i].ID < items[j].ID
		}
		return items[i].ChargeSlot < items[j].ChargeSlot
	})
	return prepareBillingPriceItemsForEditor(items)
}

func pricingItemsForGeminiRecord(record *modelCatalogRecord, layer string, rules []BillingRule) []BillingPriceItem {
	matrix := buildGeminiMatrixForRecord(record, layer, rules)
	pricing := compactGeminiPricingFromMatrix(matrix, record)
	if pricing == nil {
		pricing = selectGeminiMatrixPricing(record, layer)
	}

	items := make([]BillingPriceItem, 0, 24)
	items = append(items, pricingItemsFromFlatPricing(record, layer, pricing)...)
	items = append(items, pricingItemsFromRules(record, layer, rules)...)
	items = append(items, pricingItemsFromGeminiMatrixDiff(record, layer, matrix, pricing)...)
	return items
}

func pricingItemsFromFlatPricing(record *modelCatalogRecord, layer string, pricing *ModelCatalogPricing) []BillingPriceItem {
	if record == nil || pricing == nil {
		return []BillingPriceItem{}
	}
	explicit := billingFlatPriceExplicitFields(record, layer)
	items := make([]BillingPriceItem, 0, 16)
	appendBase := func(chargeSlot, unit string, base *float64, baseExplicit bool, priority *float64, priorityExplicit bool, threshold *int, above *float64, priorityAbove *float64) {
		if shouldExposeFlatPrice(base, baseExplicit) {
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
		if shouldExposeFlatPrice(priority, priorityExplicit) {
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

	appendBase(BillingChargeSlotTextInput, BillingUnitInputToken, pricing.InputCostPerToken, explicit["input"], pricing.InputCostPerTokenPriority, explicit["input_priority"], pricing.InputTokenThreshold, pricing.InputCostPerTokenAboveThreshold, pricing.InputCostPerTokenPriorityAboveThreshold)
	appendBase(BillingChargeSlotTextOutput, BillingUnitOutputToken, pricing.OutputCostPerToken, explicit["output"], pricing.OutputCostPerTokenPriority, explicit["output_priority"], pricing.OutputTokenThreshold, pricing.OutputCostPerTokenAboveThreshold, pricing.OutputCostPerTokenPriorityAboveThreshold)
	appendBase(BillingChargeSlotCacheCreate, BillingUnitCacheCreateToken, pricing.CacheCreationInputTokenCost, explicit["cache_create"], nil, false, nil, nil, nil)
	appendBase(BillingChargeSlotCacheRead, BillingUnitCacheReadToken, pricing.CacheReadInputTokenCost, explicit["cache_read"], pricing.CacheReadInputTokenCostPriority, explicit["cache_read_priority"], nil, nil, nil)
	appendBase(BillingChargeSlotImageOutput, BillingUnitImage, pricing.OutputCostPerImage, explicit["image_output"], pricing.OutputCostPerImagePriority, explicit["image_output_priority"], nil, nil, nil)
	appendBase(BillingChargeSlotVideoRequest, BillingUnitVideoRequest, pricing.OutputCostPerVideoRequest, explicit["video_request"], nil, false, nil, nil, nil)

	if shouldExposeFlatPrice(pricing.CacheCreationInputTokenCostAbove1hr, explicit["cache_storage"]) {
		items = append(items, BillingPriceItem{
			ID:            billingBaseItemID(layer, BillingChargeSlotCacheStorageTokenHour, ""),
			ChargeSlot:    BillingChargeSlotCacheStorageTokenHour,
			Unit:          BillingUnitCacheStorageTokenHour,
			Layer:         normalizeBillingDimension(layer, BillingLayerSale),
			Mode:          BillingPriceItemModeProviderRule,
			OperationType: "cache_storage",
			Price:         *pricing.CacheCreationInputTokenCostAbove1hr,
			Enabled:       true,
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
			ID:                rule.ID,
			ChargeSlot:        slot,
			Unit:              rule.Unit,
			Layer:             rule.Layer,
			Mode:              mode,
			ServiceTier:       rule.ServiceTier,
			BatchMode:         rule.BatchMode,
			Surface:           rule.Surface,
			OperationType:     rule.OperationType,
			InputModality:     rule.Matchers.InputModality,
			OutputModality:    rule.Matchers.OutputModality,
			CachePhase:        rule.Matchers.CachePhase,
			GroundingKind:     rule.Matchers.GroundingKind,
			ContextWindow:     rule.Matchers.ContextWindow,
			Price:             rule.Price,
			FormulaSource:     rule.FormulaSource,
			FormulaMultiplier: cloneBillingFloat64(rule.FormulaMultiplier),
			RuleID:            rule.ID,
			Enabled:           rule.Enabled,
		})
	}
	return items
}

func compactGeminiPricingFromMatrix(matrix *GeminiBillingMatrix, record *modelCatalogRecord) *ModelCatalogPricing {
	if matrix == nil {
		return nil
	}
	pricing := &ModelCatalogPricing{}

	assign := func(target **float64, surface, tier, slot string) bool {
		cell := geminiMatrixCell(matrix, surface, tier, slot)
		if cell == nil || cell.Price == nil {
			return false
		}
		*target = modelCatalogFloat64Ptr(*cell.Price)
		return true
	}

	assign(&pricing.InputCostPerToken, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotTextInput)
	assign(&pricing.InputCostPerTokenPriority, BillingSurfaceGeminiNative, BillingServiceTierPriority, BillingChargeSlotTextInput)
	assign(&pricing.OutputCostPerToken, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotTextOutput)
	assign(&pricing.OutputCostPerTokenPriority, BillingSurfaceGeminiNative, BillingServiceTierPriority, BillingChargeSlotTextOutput)
	assign(&pricing.CacheCreationInputTokenCost, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotCacheCreate)
	assign(&pricing.CacheReadInputTokenCost, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotCacheRead)
	assign(&pricing.CacheReadInputTokenCostPriority, BillingSurfaceGeminiNative, BillingServiceTierPriority, BillingChargeSlotCacheRead)
	assign(&pricing.CacheCreationInputTokenCostAbove1hr, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotCacheStorageTokenHour)
	assign(&pricing.OutputCostPerImage, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotImageOutput)
	assign(&pricing.OutputCostPerImagePriority, BillingSurfaceGeminiNative, BillingServiceTierPriority, BillingChargeSlotImageOutput)
	assign(&pricing.OutputCostPerVideoRequest, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotVideoRequest)

	if record != nil && record.longContextInputTokenThreshold > 0 {
		hasLongInput := assign(&pricing.InputCostPerTokenAboveThreshold, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotTextInputLongContext)
		hasLongPriority := assign(&pricing.InputCostPerTokenPriorityAboveThreshold, BillingSurfaceGeminiNative, BillingServiceTierPriority, BillingChargeSlotTextInputLongContext)
		if hasLongInput || hasLongPriority {
			pricing.InputTokenThreshold = modelCatalogIntPtr(record.longContextInputTokenThreshold)
		}

		hasLongOutput := assign(&pricing.OutputCostPerTokenAboveThreshold, BillingSurfaceGeminiNative, BillingServiceTierStandard, BillingChargeSlotTextOutputLongContext)
		hasLongOutputPriority := assign(&pricing.OutputCostPerTokenPriorityAboveThreshold, BillingSurfaceGeminiNative, BillingServiceTierPriority, BillingChargeSlotTextOutputLongContext)
		if hasLongOutput || hasLongOutputPriority {
			pricing.OutputTokenThreshold = modelCatalogIntPtr(record.longContextInputTokenThreshold)
		}
	}

	if pricingEmpty(pricing) {
		return nil
	}
	return pricing
}

func pricingItemsFromGeminiMatrixDiff(record *modelCatalogRecord, layer string, matrix *GeminiBillingMatrix, pricing *ModelCatalogPricing) []BillingPriceItem {
	if matrix == nil {
		return []BillingPriceItem{}
	}
	baseline := newGeminiBillingMatrix()
	if pricing != nil {
		applyPricingToGeminiMatrix(baseline, pricing, record, "editor_baseline")
		deriveGeminiMatrixAudioAndStorage(baseline, pricing)
	}

	items := make([]BillingPriceItem, 0, 16)
	for _, row := range matrix.Rows {
		for slot, cell := range row.Slots {
			if cell.Price == nil {
				continue
			}
			spec, ok := geminiMatrixSlotSpecs[slot]
			if !ok {
				continue
			}
			if geminiMatrixPriceMatchesBaseline(baseline, row.Surface, row.ServiceTier, slot, *cell.Price) {
				continue
			}

			mode := BillingPriceItemModeProviderRule
			surface := row.Surface
			if geminiMatrixCanUseServiceTierMode(row.Surface, row.ServiceTier, slot) {
				mode = BillingPriceItemModeServiceTier
				surface = ""
			}

			items = append(items, BillingPriceItem{
				ID:             geminiMatrixRuleID(record.model, layer, row.Surface, row.ServiceTier, slot),
				ChargeSlot:     slot,
				Unit:           spec.unit,
				Layer:          normalizeBillingDimension(layer, BillingLayerSale),
				Mode:           mode,
				ServiceTier:    row.ServiceTier,
				Surface:        surface,
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

func geminiMatrixCanUseServiceTierMode(surface string, serviceTier string, slot string) bool {
	if normalizeBillingSurface(surface) != BillingSurfaceGeminiNative {
		return false
	}
	switch normalizeBillingServiceTier(serviceTier) {
	case BillingServiceTierFlex, BillingServiceTierPriority:
	default:
		return false
	}
	switch normalizeBillingDimension(slot, "") {
	case BillingChargeSlotTextInput, BillingChargeSlotTextOutput, BillingChargeSlotCacheRead, BillingChargeSlotImageOutput:
		return true
	default:
		return false
	}
}

func geminiMatrixPriceMatchesBaseline(matrix *GeminiBillingMatrix, surface string, tier string, slot string, actual float64) bool {
	cell := geminiMatrixCell(matrix, surface, tier, slot)
	if cell == nil || cell.Price == nil {
		return false
	}
	return billingPricesAlmostEqual(actual, *cell.Price)
}

func billingPricesAlmostEqual(left float64, right float64) bool {
	delta := math.Abs(left - right)
	scale := math.Max(1, math.Max(math.Abs(left), math.Abs(right)))
	return delta <= 1e-12*scale
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

func prepareBillingPriceItemsForEditor(items []BillingPriceItem) []BillingPriceItem {
	deduped := dedupeBillingPriceItems(items)
	if len(deduped) == 0 {
		return []BillingPriceItem{}
	}
	for index := range deduped {
		deduped[index] = sanitizeBillingPriceItemForEditor(deduped[index])
	}
	return deduped
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
	item.FormulaSource = normalizeBillingDimension(item.FormulaSource, "")
	item.FormulaMultiplier = cloneBillingFloat64(item.FormulaMultiplier)
	if item.ID == "" {
		item.ID = billingGeneratedRuleID(item)
	}
	item.Enabled = item.Enabled || item.Price > 0
	return item
}

func sanitizeBillingPriceItemForEditor(item BillingPriceItem) BillingPriceItem {
	item = normalizeBillingPriceItem(item)
	if item.ServiceTier == BillingServiceTierStandard {
		item.ServiceTier = ""
	}
	if item.BatchMode == BillingBatchModeAny {
		item.BatchMode = ""
	}
	if item.Surface == BillingSurfaceGeminiNative || item.Surface == BillingSurfaceAny {
		item.Surface = ""
	}
	if item.OperationType == operationTypeForChargeSlot(item.ChargeSlot) {
		item.OperationType = ""
	}
	return item
}

func shouldExposeFlatPrice(value *float64, explicit bool) bool {
	if value == nil {
		return false
	}
	return explicit || math.Abs(*value) > 0
}

func billingFlatPriceExplicitFields(record *modelCatalogRecord, layer string) map[string]bool {
	explicit := map[string]bool{}
	if record == nil {
		return explicit
	}
	var override *ModelPricingOverride
	switch normalizeBillingDimension(layer, BillingLayerSale) {
	case BillingLayerOfficial:
		override = record.officialOverridePricing
	case BillingLayerSale:
		override = record.saleOverridePricing
	}
	if override == nil {
		return explicit
	}
	explicit["input"] = override.InputCostPerToken != nil
	explicit["input_priority"] = override.InputCostPerTokenPriority != nil
	explicit["output"] = override.OutputCostPerToken != nil
	explicit["output_priority"] = override.OutputCostPerTokenPriority != nil
	explicit["cache_create"] = override.CacheCreationInputTokenCost != nil
	explicit["cache_read"] = override.CacheReadInputTokenCost != nil
	explicit["cache_read_priority"] = override.CacheReadInputTokenCostPriority != nil
	explicit["cache_storage"] = override.CacheCreationInputTokenCostAbove1hr != nil
	explicit["image_output"] = override.OutputCostPerImage != nil
	explicit["image_output_priority"] = override.OutputCostPerImagePriority != nil
	explicit["video_request"] = override.OutputCostPerVideoRequest != nil
	return explicit
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
