package service

import (
	"math"
	"strings"

	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

type billingPricingFormMetadata struct {
	InputSupported   bool
	OutputChargeSlot string
}

const (
	billingDiscountFieldInputPrice                = "input_price"
	billingDiscountFieldOutputPrice               = "output_price"
	billingDiscountFieldCachePrice                = "cache_price"
	billingDiscountFieldInputPriceAboveThreshold  = "input_price_above_threshold"
	billingDiscountFieldOutputPriceAboveThreshold = "output_price_above_threshold"
	billingDiscountFieldBatchInputPrice           = "batch_input_price"
	billingDiscountFieldBatchOutputPrice          = "batch_output_price"
	billingDiscountFieldBatchCachePrice           = "batch_cache_price"
	billingDiscountFieldGroundingSearch           = "grounding_search"
	billingDiscountFieldGroundingMaps             = "grounding_maps"
	billingDiscountFieldFileSearchEmbedding       = "file_search_embedding"
	billingDiscountFieldFileSearchRetrieval       = "file_search_retrieval"
)

func billingPricingMetadataForRecord(record *modelCatalogRecord, items []BillingPriceItem) billingPricingFormMetadata {
	outputSlot := billingDefaultOutputChargeSlot("")
	if record != nil {
		outputSlot = billingDefaultOutputChargeSlot(record.mode)
	}
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch {
			continue
		}
		if billingPricingUsesLegacyTier(item) {
			continue
		}
		switch item.ChargeSlot {
		case BillingChargeSlotTextOutput, BillingChargeSlotImageOutput, BillingChargeSlotVideoRequest:
			outputSlot = item.ChargeSlot
		}
	}
	inputSupported := outputSlot != BillingChargeSlotVideoRequest
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if item.ChargeSlot == BillingChargeSlotTextInput {
			inputSupported = true
			break
		}
	}
	return billingPricingFormMetadata{
		InputSupported:   inputSupported,
		OutputChargeSlot: outputSlot,
	}
}

func billingDefaultOutputChargeSlot(mode string) string {
	normalized := strings.ToLower(strings.TrimSpace(mode))
	switch {
	case strings.Contains(normalized, "video"):
		return BillingChargeSlotVideoRequest
	case strings.Contains(normalized, "image"):
		return BillingChargeSlotImageOutput
	default:
		return BillingChargeSlotTextOutput
	}
}

func billingPricingLayerFormFromItemsWithMetadata(metadata billingPricingFormMetadata, items []BillingPriceItem) BillingPricingLayerForm {
	form := BillingPricingLayerForm{}
	if metadata.InputSupported {
		form.InputPrice = billingPricingSingleSlotValue(items, BillingChargeSlotTextInput, false)
		if threshold, price := billingPricingTierValue(items, BillingChargeSlotTextInput); threshold != nil && price != nil {
			form.TieredEnabled = true
			form.TierThresholdTokens = threshold
			form.InputPriceAboveThreshold = price
		}
	}
	if outputSlot := metadata.OutputChargeSlot; outputSlot != "" {
		form.OutputPrice = billingPricingSingleSlotValue(items, outputSlot, false)
		if outputSlot == BillingChargeSlotTextOutput {
			if threshold, price := billingPricingTierValue(items, BillingChargeSlotTextOutput); threshold != nil && price != nil {
				form.TieredEnabled = true
				if form.TierThresholdTokens == nil {
					form.TierThresholdTokens = threshold
				}
				form.OutputPriceAboveThreshold = price
			}
		}
	}
	form.CachePrice = billingPricingMaxSlotValue(items, false,
		BillingChargeSlotCacheCreate,
		BillingChargeSlotCacheRead,
		BillingChargeSlotCacheStorageTokenHour,
	)
	form.Special.BatchInputPrice = billingPricingSingleSlotValue(items, BillingChargeSlotTextInput, true)
	form.Special.BatchOutputPrice = billingPricingSingleSlotValue(items, metadata.OutputChargeSlot, true)
	form.Special.BatchCachePrice = billingPricingMaxSlotValue(items, true,
		BillingChargeSlotCacheCreate,
		BillingChargeSlotCacheRead,
	)
	form.Special.GroundingSearch = billingPricingSingleSpecialValue(items, BillingChargeSlotGroundingSearchRequest)
	form.Special.GroundingMaps = billingPricingSingleSpecialValue(items, BillingChargeSlotGroundingMapsRequest)
	form.Special.FileSearchEmbedding = billingPricingSingleSpecialValue(items, BillingChargeSlotFileSearchEmbeddingToken)
	form.Special.FileSearchRetrieval = billingPricingSingleSpecialValue(items, BillingChargeSlotFileSearchRetrievalToken)
	form.SpecialEnabled = billingPricingSimpleSpecialEnabled(form.Special)
	return form
}

func billingPricingItemsFromForm(metadata billingPricingFormMetadata, layer string, form BillingPricingLayerForm) []BillingPriceItem {
	items := make([]BillingPriceItem, 0, 12)
	appendItem := func(item BillingPriceItem) {
		item.Layer = normalizeBillingDimension(layer, BillingLayerSale)
		items = append(items, normalizeBillingPriceItem(item))
	}
	appendBaseItem := func(chargeSlot string, price *float64, threshold *int, above *float64) {
		if chargeSlot == "" || price == nil {
			return
		}
		item := BillingPriceItem{
			ID:         billingBaseItemID(layer, chargeSlot, ""),
			ChargeSlot: chargeSlot,
			Unit:       billingUnitForChargeSlot(chargeSlot),
			Mode:       BillingPriceItemModeBase,
			Price:      *price,
			Enabled:    true,
		}
		if threshold != nil && above != nil {
			item.Mode = BillingPriceItemModeTiered
			item.ThresholdTokens = modelCatalogIntPtr(*threshold)
			item.PriceAboveThresh = modelCatalogFloat64Ptr(*above)
		}
		appendItem(item)
	}

	if metadata.InputSupported {
		appendBaseItem(BillingChargeSlotTextInput, form.InputPrice, billingTierThresholdForForm(form), form.InputPriceAboveThreshold)
	}
	appendBaseItem(metadata.OutputChargeSlot, form.OutputPrice, billingTierThresholdForOutput(metadata, form), form.OutputPriceAboveThreshold)
	if form.CachePrice != nil {
		appendBaseItem(BillingChargeSlotCacheCreate, form.CachePrice, nil, nil)
		appendBaseItem(BillingChargeSlotCacheRead, form.CachePrice, nil, nil)
		appendBaseItem(BillingChargeSlotCacheStorageTokenHour, form.CachePrice, nil, nil)
	}
	if !form.SpecialEnabled {
		return items
	}
	if metadata.InputSupported && form.Special.BatchInputPrice != nil {
		appendItem(BillingPriceItem{
			ID:         billingBaseItemID(layer, BillingChargeSlotTextInput, BillingBatchModeBatch),
			ChargeSlot: BillingChargeSlotTextInput,
			Unit:       billingUnitForChargeSlot(BillingChargeSlotTextInput),
			Mode:       BillingPriceItemModeBatch,
			BatchMode:  BillingBatchModeBatch,
			Price:      *form.Special.BatchInputPrice,
			Enabled:    true,
		})
	}
	if metadata.OutputChargeSlot != "" && form.Special.BatchOutputPrice != nil {
		appendItem(BillingPriceItem{
			ID:         billingBaseItemID(layer, metadata.OutputChargeSlot, BillingBatchModeBatch),
			ChargeSlot: metadata.OutputChargeSlot,
			Unit:       billingUnitForChargeSlot(metadata.OutputChargeSlot),
			Mode:       BillingPriceItemModeBatch,
			BatchMode:  BillingBatchModeBatch,
			Price:      *form.Special.BatchOutputPrice,
			Enabled:    true,
		})
	}
	if form.Special.BatchCachePrice != nil {
		appendItem(BillingPriceItem{
			ID:         billingBaseItemID(layer, BillingChargeSlotCacheCreate, BillingBatchModeBatch),
			ChargeSlot: BillingChargeSlotCacheCreate,
			Unit:       billingUnitForChargeSlot(BillingChargeSlotCacheCreate),
			Mode:       BillingPriceItemModeBatch,
			BatchMode:  BillingBatchModeBatch,
			Price:      *form.Special.BatchCachePrice,
			Enabled:    true,
		})
		appendItem(BillingPriceItem{
			ID:         billingBaseItemID(layer, BillingChargeSlotCacheRead, BillingBatchModeBatch),
			ChargeSlot: BillingChargeSlotCacheRead,
			Unit:       billingUnitForChargeSlot(BillingChargeSlotCacheRead),
			Mode:       BillingPriceItemModeBatch,
			BatchMode:  BillingBatchModeBatch,
			Price:      *form.Special.BatchCachePrice,
			Enabled:    true,
		})
	}
	appendGeminiSpecialItem := func(chargeSlot string, price *float64) {
		if price == nil {
			return
		}
		appendItem(BillingPriceItem{
			ID:         billingBaseItemID(layer, chargeSlot, "special"),
			ChargeSlot: chargeSlot,
			Unit:       billingUnitForChargeSlot(chargeSlot),
			Mode:       BillingPriceItemModeProviderRule,
			Surface:    BillingSurfaceGeminiNative,
			Price:      *price,
			Enabled:    true,
		})
	}
	appendGeminiSpecialItem(BillingChargeSlotGroundingSearchRequest, form.Special.GroundingSearch)
	appendGeminiSpecialItem(BillingChargeSlotGroundingMapsRequest, form.Special.GroundingMaps)
	appendGeminiSpecialItem(BillingChargeSlotFileSearchEmbeddingToken, form.Special.FileSearchEmbedding)
	appendGeminiSpecialItem(BillingChargeSlotFileSearchRetrievalToken, form.Special.FileSearchRetrieval)
	return items
}

func billingPricingRulesFromForm(record *modelCatalogRecord, layer string, items []BillingPriceItem) []BillingRule {
	if !isGeminiBillingCompatModel(record.model) {
		return pricingRulesFromItems(record, layer, items)
	}
	filtered := make([]BillingPriceItem, 0, len(items))
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if billingPricingIsGeminiMatrixSpecialItem(item) {
			continue
		}
		filtered = append(filtered, item)
	}
	return pricingRulesFromItems(record, layer, filtered)
}

func geminiMatrixFromSimpleForm(form BillingPricingLayerForm) *GeminiBillingMatrix {
	matrix := newGeminiBillingMatrix()
	appendCell := func(slot string, price *float64) {
		if price == nil {
			return
		}
		setGeminiMatrixCell(matrix, BillingSurfaceGeminiNative, BillingServiceTierStandard, slot, price, "", "simple_form", true)
	}
	appendCell(BillingChargeSlotGroundingSearchRequest, form.Special.GroundingSearch)
	appendCell(BillingChargeSlotGroundingMapsRequest, form.Special.GroundingMaps)
	appendCell(BillingChargeSlotFileSearchEmbeddingToken, form.Special.FileSearchEmbedding)
	appendCell(BillingChargeSlotFileSearchRetrievalToken, form.Special.FileSearchRetrieval)
	return matrix
}

func cloneBillingPricingLayerForm(form BillingPricingLayerForm) BillingPricingLayerForm {
	cloned := BillingPricingLayerForm{
		SpecialEnabled: form.SpecialEnabled,
		TieredEnabled:  form.TieredEnabled,
		Special: BillingPricingSimpleSpecial{
			BatchInputPrice:     cloneBillingFloat64(form.Special.BatchInputPrice),
			BatchOutputPrice:    cloneBillingFloat64(form.Special.BatchOutputPrice),
			BatchCachePrice:     cloneBillingFloat64(form.Special.BatchCachePrice),
			GroundingSearch:     cloneBillingFloat64(form.Special.GroundingSearch),
			GroundingMaps:       cloneBillingFloat64(form.Special.GroundingMaps),
			FileSearchEmbedding: cloneBillingFloat64(form.Special.FileSearchEmbedding),
			FileSearchRetrieval: cloneBillingFloat64(form.Special.FileSearchRetrieval),
		},
	}
	cloned.InputPrice = cloneBillingFloat64(form.InputPrice)
	cloned.OutputPrice = cloneBillingFloat64(form.OutputPrice)
	cloned.CachePrice = cloneBillingFloat64(form.CachePrice)
	cloned.TierThresholdTokens = cloneBillingInt(form.TierThresholdTokens)
	cloned.InputPriceAboveThreshold = cloneBillingFloat64(form.InputPriceAboveThreshold)
	cloned.OutputPriceAboveThreshold = cloneBillingFloat64(form.OutputPriceAboveThreshold)
	return cloned
}

func validateBillingPricingLayerForm(form BillingPricingLayerForm) error {
	values := []*float64{
		form.InputPrice,
		form.OutputPrice,
		form.CachePrice,
		form.InputPriceAboveThreshold,
		form.OutputPriceAboveThreshold,
		form.Special.BatchInputPrice,
		form.Special.BatchOutputPrice,
		form.Special.BatchCachePrice,
		form.Special.GroundingSearch,
		form.Special.GroundingMaps,
		form.Special.FileSearchEmbedding,
		form.Special.FileSearchRetrieval,
	}
	for _, value := range values {
		if value == nil {
			continue
		}
		if math.IsNaN(*value) || math.IsInf(*value, 0) || *value < 0 {
			return infraerrors.BadRequest("BILLING_PRICE_INVALID", "pricing must be a non-negative number")
		}
	}
	if form.TierThresholdTokens != nil && *form.TierThresholdTokens <= 0 {
		return infraerrors.BadRequest("BILLING_PRICE_INVALID", "tier threshold must be a positive integer")
	}
	if !form.TieredEnabled {
		return nil
	}
	if form.TierThresholdTokens == nil {
		return infraerrors.BadRequest("BILLING_PRICE_INVALID", "tiered pricing requires a shared threshold")
	}
	if form.InputPriceAboveThreshold == nil && form.OutputPriceAboveThreshold == nil {
		return infraerrors.BadRequest("BILLING_PRICE_INVALID", "tiered pricing requires at least one above-threshold price")
	}
	return nil
}

func applyDiscountToBillingPricingLayerForm(form BillingPricingLayerForm, ratio float64, selected map[string]struct{}) BillingPricingLayerForm {
	next := cloneBillingPricingLayerForm(form)
	discount := func(id string, value **float64) {
		if *value == nil {
			return
		}
		if len(selected) > 0 {
			if _, ok := selected[id]; !ok {
				return
			}
		}
		*value = modelCatalogFloat64Ptr(**value * ratio)
	}
	discount(billingDiscountFieldInputPrice, &next.InputPrice)
	discount(billingDiscountFieldOutputPrice, &next.OutputPrice)
	discount(billingDiscountFieldCachePrice, &next.CachePrice)
	discount(billingDiscountFieldInputPriceAboveThreshold, &next.InputPriceAboveThreshold)
	discount(billingDiscountFieldOutputPriceAboveThreshold, &next.OutputPriceAboveThreshold)
	discount(billingDiscountFieldBatchInputPrice, &next.Special.BatchInputPrice)
	discount(billingDiscountFieldBatchOutputPrice, &next.Special.BatchOutputPrice)
	discount(billingDiscountFieldBatchCachePrice, &next.Special.BatchCachePrice)
	discount(billingDiscountFieldGroundingSearch, &next.Special.GroundingSearch)
	discount(billingDiscountFieldGroundingMaps, &next.Special.GroundingMaps)
	discount(billingDiscountFieldFileSearchEmbedding, &next.Special.FileSearchEmbedding)
	discount(billingDiscountFieldFileSearchRetrieval, &next.Special.FileSearchRetrieval)
	next.SpecialEnabled = billingPricingSimpleSpecialEnabled(next.Special)
	return next
}

func billingPricingSingleSlotValue(items []BillingPriceItem, slot string, batch bool) *float64 {
	if slot == "" {
		return nil
	}
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if item.ChargeSlot != normalizeBillingDimension(slot, "") ||
			(normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch) != batch ||
			billingPricingUsesLegacyTier(item) {
			continue
		}
		return modelCatalogFloat64Ptr(item.Price)
	}
	return nil
}

func billingPricingTierValue(items []BillingPriceItem, slot string) (*int, *float64) {
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if item.ChargeSlot != normalizeBillingDimension(slot, "") ||
			normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch ||
			billingPricingUsesLegacyTier(item) ||
			item.ThresholdTokens == nil ||
			item.PriceAboveThresh == nil {
			continue
		}
		return modelCatalogIntPtr(*item.ThresholdTokens), modelCatalogFloat64Ptr(*item.PriceAboveThresh)
	}
	return nil, nil
}

func billingPricingMaxSlotValue(items []BillingPriceItem, batch bool, slots ...string) *float64 {
	var (
		found bool
		max   float64
	)
	allowed := make(map[string]struct{}, len(slots))
	for _, slot := range slots {
		allowed[normalizeBillingDimension(slot, "")] = struct{}{}
	}
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if _, ok := allowed[item.ChargeSlot]; !ok ||
			(normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch) != batch ||
			billingPricingUsesLegacyTier(item) {
			continue
		}
		if !found || item.Price > max {
			max = item.Price
			found = true
		}
	}
	if !found {
		return nil
	}
	return modelCatalogFloat64Ptr(max)
}

func billingPricingSingleSpecialValue(items []BillingPriceItem, slot string) *float64 {
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if item.ChargeSlot != normalizeBillingDimension(slot, "") ||
			normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch ||
			billingPricingUsesLegacyTier(item) {
			continue
		}
		return modelCatalogFloat64Ptr(item.Price)
	}
	return nil
}

func billingPricingUsesLegacyTier(item BillingPriceItem) bool {
	tier := normalizeBillingServiceTier(item.ServiceTier)
	return tier == BillingServiceTierPriority || tier == BillingServiceTierFlex
}

func billingPricingSimpleSpecialEnabled(special BillingPricingSimpleSpecial) bool {
	return special.BatchInputPrice != nil ||
		special.BatchOutputPrice != nil ||
		special.BatchCachePrice != nil ||
		special.GroundingSearch != nil ||
		special.GroundingMaps != nil ||
		special.FileSearchEmbedding != nil ||
		special.FileSearchRetrieval != nil
}

func billingPricingIsGeminiMatrixSpecialItem(item BillingPriceItem) bool {
	if normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch {
		return false
	}
	switch item.ChargeSlot {
	case BillingChargeSlotGroundingSearchRequest,
		BillingChargeSlotGroundingMapsRequest,
		BillingChargeSlotFileSearchEmbeddingToken,
		BillingChargeSlotFileSearchRetrievalToken:
		return true
	default:
		return false
	}
}

func billingUnitForChargeSlot(chargeSlot string) string {
	switch normalizeBillingDimension(chargeSlot, "") {
	case BillingChargeSlotTextInput:
		return BillingUnitInputToken
	case BillingChargeSlotTextOutput:
		return BillingUnitOutputToken
	case BillingChargeSlotCacheCreate:
		return BillingUnitCacheCreateToken
	case BillingChargeSlotCacheRead:
		return BillingUnitCacheReadToken
	case BillingChargeSlotCacheStorageTokenHour:
		return BillingUnitCacheStorageTokenHour
	case BillingChargeSlotImageOutput:
		return BillingUnitImage
	case BillingChargeSlotVideoRequest:
		return BillingUnitVideoRequest
	case BillingChargeSlotGroundingSearchRequest:
		return BillingUnitGroundingSearchRequest
	case BillingChargeSlotGroundingMapsRequest:
		return BillingUnitGroundingMapsRequest
	case BillingChargeSlotFileSearchEmbeddingToken:
		return BillingUnitFileSearchEmbedding
	case BillingChargeSlotFileSearchRetrievalToken:
		return BillingUnitFileSearchRetrieval
	default:
		return BillingUnitInputToken
	}
}

func billingTierThresholdForForm(form BillingPricingLayerForm) *int {
	if !form.TieredEnabled {
		return nil
	}
	if form.InputPriceAboveThreshold == nil || form.TierThresholdTokens == nil {
		return nil
	}
	return modelCatalogIntPtr(*form.TierThresholdTokens)
}

func billingTierThresholdForOutput(metadata billingPricingFormMetadata, form BillingPricingLayerForm) *int {
	if !form.TieredEnabled || metadata.OutputChargeSlot != BillingChargeSlotTextOutput {
		return nil
	}
	if form.OutputPriceAboveThreshold == nil || form.TierThresholdTokens == nil {
		return nil
	}
	return modelCatalogIntPtr(*form.TierThresholdTokens)
}

func cloneBillingFloat64(value *float64) *float64 {
	if value == nil {
		return nil
	}
	return modelCatalogFloat64Ptr(*value)
}

func cloneBillingInt(value *int) *int {
	if value == nil {
		return nil
	}
	return modelCatalogIntPtr(*value)
}
