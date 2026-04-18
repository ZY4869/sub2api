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
	billingPricingFormulaSourceShared             = "sale_multiplier_shared"
	billingPricingFormulaSourceItem               = "sale_multiplier_item"
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
	billingPricingApplyMultiplierConfig(&form, metadata, items)
	return form
}

func billingPricingItemsFromForm(metadata billingPricingFormMetadata, layer string, form BillingPricingLayerForm) []BillingPriceItem {
	form = normalizeBillingPricingLayerFormForLayer(form, layer)
	items := make([]BillingPriceItem, 0, 12)
	appendItem := func(fieldID string, item BillingPriceItem) {
		item.Layer = normalizeBillingDimension(layer, BillingLayerSale)
		billingPricingApplyFormulaMetadata(&item, form, fieldID)
		items = append(items, normalizeBillingPriceItem(item))
	}
	appendBaseItem := func(fieldID string, chargeSlot string, price *float64, threshold *int, above *float64) {
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
		appendItem(fieldID, item)
	}

	if metadata.InputSupported {
		appendBaseItem(billingDiscountFieldInputPrice, BillingChargeSlotTextInput, form.InputPrice, billingTierThresholdForForm(form), form.InputPriceAboveThreshold)
	}
	appendBaseItem(billingDiscountFieldOutputPrice, metadata.OutputChargeSlot, form.OutputPrice, billingTierThresholdForOutput(metadata, form), form.OutputPriceAboveThreshold)
	if form.CachePrice != nil {
		appendBaseItem(billingDiscountFieldCachePrice, BillingChargeSlotCacheCreate, form.CachePrice, nil, nil)
		appendBaseItem(billingDiscountFieldCachePrice, BillingChargeSlotCacheRead, form.CachePrice, nil, nil)
		appendBaseItem(billingDiscountFieldCachePrice, BillingChargeSlotCacheStorageTokenHour, form.CachePrice, nil, nil)
	}
	if !form.SpecialEnabled {
		return items
	}
	if metadata.InputSupported && form.Special.BatchInputPrice != nil {
		appendItem(billingDiscountFieldBatchInputPrice, BillingPriceItem{
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
		appendItem(billingDiscountFieldBatchOutputPrice, BillingPriceItem{
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
		appendItem(billingDiscountFieldBatchCachePrice, BillingPriceItem{
			ID:         billingBaseItemID(layer, BillingChargeSlotCacheCreate, BillingBatchModeBatch),
			ChargeSlot: BillingChargeSlotCacheCreate,
			Unit:       billingUnitForChargeSlot(BillingChargeSlotCacheCreate),
			Mode:       BillingPriceItemModeBatch,
			BatchMode:  BillingBatchModeBatch,
			Price:      *form.Special.BatchCachePrice,
			Enabled:    true,
		})
		appendItem(billingDiscountFieldBatchCachePrice, BillingPriceItem{
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
		appendItem(billingPricingFieldIDForSpecialSlot(chargeSlot), BillingPriceItem{
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
		SpecialEnabled:    form.SpecialEnabled,
		TieredEnabled:     form.TieredEnabled,
		MultiplierEnabled: form.MultiplierEnabled,
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
	cloned.SharedMultiplier = cloneBillingFloat64(form.SharedMultiplier)
	cloned.ItemMultipliers = cloneBillingMultiplierMap(form.ItemMultipliers)
	if form.MultiplierEnabled {
		cloned.MultiplierMode = normalizeBillingPricingMultiplierMode(form.MultiplierMode)
	}
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
	validateMultiplier := func(value float64) error {
		if math.IsNaN(value) || math.IsInf(value, 0) || value < 0 {
			return infraerrors.BadRequest("BILLING_PRICE_INVALID", "multiplier must be a non-negative number")
		}
		return nil
	}
	if form.SharedMultiplier != nil {
		if err := validateMultiplier(*form.SharedMultiplier); err != nil {
			return err
		}
	}
	for _, value := range form.ItemMultipliers {
		if err := validateMultiplier(value); err != nil {
			return err
		}
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

func normalizeBillingPricingLayerFormForLayer(form BillingPricingLayerForm, layer string) BillingPricingLayerForm {
	next := cloneBillingPricingLayerForm(form)
	next.MultiplierMode = normalizeBillingPricingMultiplierMode(next.MultiplierMode)
	if normalizeBillingDimension(layer, BillingLayerSale) != BillingLayerSale {
		next.MultiplierEnabled = false
		next.MultiplierMode = ""
		next.SharedMultiplier = nil
		next.ItemMultipliers = nil
		return next
	}
	if !next.MultiplierEnabled {
		next.MultiplierMode = ""
		next.SharedMultiplier = nil
		next.ItemMultipliers = nil
		return next
	}
	if next.MultiplierMode == "" {
		next.MultiplierMode = BillingPricingMultiplierShared
	}
	if next.MultiplierMode == BillingPricingMultiplierShared {
		next.ItemMultipliers = nil
	}
	return next
}

func normalizeBillingPricingMultiplierMode(mode BillingPricingMultiplierMode) BillingPricingMultiplierMode {
	switch BillingPricingMultiplierMode(strings.TrimSpace(strings.ToLower(string(mode)))) {
	case BillingPricingMultiplierItem:
		return BillingPricingMultiplierItem
	default:
		return BillingPricingMultiplierShared
	}
}

func cloneBillingMultiplierMap(values map[string]float64) map[string]float64 {
	if len(values) == 0 {
		return nil
	}
	cloned := make(map[string]float64, len(values))
	for key, value := range values {
		trimmed := strings.TrimSpace(key)
		if trimmed == "" {
			continue
		}
		cloned[trimmed] = value
	}
	if len(cloned) == 0 {
		return nil
	}
	return cloned
}

func billingPricingApplyFormulaMetadata(item *BillingPriceItem, form BillingPricingLayerForm, fieldID string) {
	if item == nil || !form.MultiplierEnabled || fieldID == "" {
		return
	}
	multiplier := billingPricingMultiplierForField(form, fieldID)
	if multiplier == nil {
		return
	}
	item.FormulaMultiplier = multiplier
	item.FormulaSource = billingPricingFormulaSource(form.MultiplierMode)
}

func billingPricingFormulaSource(mode BillingPricingMultiplierMode) string {
	if normalizeBillingPricingMultiplierMode(mode) == BillingPricingMultiplierItem {
		return billingPricingFormulaSourceItem
	}
	return billingPricingFormulaSourceShared
}

func billingPricingMultiplierForField(form BillingPricingLayerForm, fieldID string) *float64 {
	if !form.MultiplierEnabled || strings.TrimSpace(fieldID) == "" {
		return nil
	}
	if normalizeBillingPricingMultiplierMode(form.MultiplierMode) == BillingPricingMultiplierItem {
		if value, ok := form.ItemMultipliers[fieldID]; ok {
			return modelCatalogFloat64Ptr(value)
		}
		return modelCatalogFloat64Ptr(1)
	}
	if form.SharedMultiplier != nil {
		return cloneBillingFloat64(form.SharedMultiplier)
	}
	return modelCatalogFloat64Ptr(1)
}

func billingPricingApplyMultiplierConfig(form *BillingPricingLayerForm, metadata billingPricingFormMetadata, items []BillingPriceItem) {
	if form == nil {
		return
	}
	fieldMultipliers := map[string]float64{}
	for _, raw := range items {
		item := normalizeBillingPriceItem(raw)
		if item.FormulaMultiplier == nil {
			continue
		}
		fieldID := billingPricingFieldIDForItem(metadata, item)
		if fieldID == "" {
			continue
		}
		fieldMultipliers[fieldID] = *item.FormulaMultiplier
		if item.Mode == BillingPriceItemModeTiered {
			switch fieldID {
			case billingDiscountFieldInputPrice:
				if form.InputPriceAboveThreshold != nil {
					fieldMultipliers[billingDiscountFieldInputPriceAboveThreshold] = *item.FormulaMultiplier
				}
			case billingDiscountFieldOutputPrice:
				if form.OutputPriceAboveThreshold != nil {
					fieldMultipliers[billingDiscountFieldOutputPriceAboveThreshold] = *item.FormulaMultiplier
				}
			}
		}
	}
	if len(fieldMultipliers) == 0 {
		return
	}
	form.MultiplierEnabled = true
	var (
		uniformValue float64
		uniformSet   bool
		mixed        bool
	)
	for _, value := range fieldMultipliers {
		if !uniformSet {
			uniformValue = value
			uniformSet = true
			continue
		}
		if !billingPricesAlmostEqual(uniformValue, value) {
			mixed = true
			break
		}
	}
	if !mixed && uniformSet {
		form.MultiplierMode = BillingPricingMultiplierShared
		form.SharedMultiplier = modelCatalogFloat64Ptr(uniformValue)
		form.ItemMultipliers = nil
		return
	}
	form.MultiplierMode = BillingPricingMultiplierItem
	form.SharedMultiplier = nil
	form.ItemMultipliers = fieldMultipliers
}

func billingPricingFieldIDForItem(metadata billingPricingFormMetadata, item BillingPriceItem) string {
	batch := normalizeBillingActualBatchMode(item.BatchMode) == BillingBatchModeBatch
	switch item.ChargeSlot {
	case BillingChargeSlotTextInput:
		if batch {
			return billingDiscountFieldBatchInputPrice
		}
		return billingDiscountFieldInputPrice
	case BillingChargeSlotTextOutput, BillingChargeSlotImageOutput, BillingChargeSlotVideoRequest:
		if batch {
			return billingDiscountFieldBatchOutputPrice
		}
		return billingDiscountFieldOutputPrice
	case BillingChargeSlotCacheCreate, BillingChargeSlotCacheRead, BillingChargeSlotCacheStorageTokenHour:
		if batch {
			return billingDiscountFieldBatchCachePrice
		}
		return billingDiscountFieldCachePrice
	case BillingChargeSlotGroundingSearchRequest:
		return billingDiscountFieldGroundingSearch
	case BillingChargeSlotGroundingMapsRequest:
		return billingDiscountFieldGroundingMaps
	case BillingChargeSlotFileSearchEmbeddingToken:
		return billingDiscountFieldFileSearchEmbedding
	case BillingChargeSlotFileSearchRetrievalToken:
		return billingDiscountFieldFileSearchRetrieval
	}
	if batch && metadata.OutputChargeSlot != "" && item.ChargeSlot == metadata.OutputChargeSlot {
		return billingDiscountFieldBatchOutputPrice
	}
	if !batch && metadata.OutputChargeSlot != "" && item.ChargeSlot == metadata.OutputChargeSlot {
		return billingDiscountFieldOutputPrice
	}
	return ""
}

func billingPricingFieldIDForSpecialSlot(slot string) string {
	switch normalizeBillingDimension(slot, "") {
	case BillingChargeSlotGroundingSearchRequest:
		return billingDiscountFieldGroundingSearch
	case BillingChargeSlotGroundingMapsRequest:
		return billingDiscountFieldGroundingMaps
	case BillingChargeSlotFileSearchEmbeddingToken:
		return billingDiscountFieldFileSearchEmbedding
	case BillingChargeSlotFileSearchRetrievalToken:
		return billingDiscountFieldFileSearchRetrieval
	default:
		return ""
	}
}

func billingPricingConfiguredFieldValues(form BillingPricingLayerForm) map[string]*float64 {
	values := map[string]*float64{
		billingDiscountFieldInputPrice:  cloneBillingFloat64(form.InputPrice),
		billingDiscountFieldOutputPrice: cloneBillingFloat64(form.OutputPrice),
		billingDiscountFieldCachePrice:  cloneBillingFloat64(form.CachePrice),
	}
	if form.TieredEnabled {
		values[billingDiscountFieldInputPriceAboveThreshold] = cloneBillingFloat64(form.InputPriceAboveThreshold)
		values[billingDiscountFieldOutputPriceAboveThreshold] = cloneBillingFloat64(form.OutputPriceAboveThreshold)
	}
	if form.SpecialEnabled {
		values[billingDiscountFieldBatchInputPrice] = cloneBillingFloat64(form.Special.BatchInputPrice)
		values[billingDiscountFieldBatchOutputPrice] = cloneBillingFloat64(form.Special.BatchOutputPrice)
		values[billingDiscountFieldBatchCachePrice] = cloneBillingFloat64(form.Special.BatchCachePrice)
		values[billingDiscountFieldGroundingSearch] = cloneBillingFloat64(form.Special.GroundingSearch)
		values[billingDiscountFieldGroundingMaps] = cloneBillingFloat64(form.Special.GroundingMaps)
		values[billingDiscountFieldFileSearchEmbedding] = cloneBillingFloat64(form.Special.FileSearchEmbedding)
		values[billingDiscountFieldFileSearchRetrieval] = cloneBillingFloat64(form.Special.FileSearchRetrieval)
	}
	for key, value := range values {
		if value == nil {
			delete(values, key)
		}
	}
	return values
}

func billingPricingEffectiveFieldValue(form BillingPricingLayerForm, fieldID string) *float64 {
	values := billingPricingConfiguredFieldValues(form)
	value, ok := values[fieldID]
	if !ok || value == nil {
		return nil
	}
	if !form.MultiplierEnabled {
		return value
	}
	multiplier := billingPricingMultiplierForField(form, fieldID)
	if multiplier == nil {
		return value
	}
	return modelCatalogFloat64Ptr(*value * *multiplier)
}

func billingPricingConfiguredMultiplierValues(form BillingPricingLayerForm) map[string]float64 {
	if !form.MultiplierEnabled {
		return nil
	}
	configured := billingPricingConfiguredFieldValues(form)
	if len(configured) == 0 {
		return nil
	}
	values := make(map[string]float64, len(configured))
	for fieldID := range configured {
		multiplier := billingPricingMultiplierForField(form, fieldID)
		if multiplier == nil {
			continue
		}
		values[fieldID] = *multiplier
	}
	if len(values) == 0 {
		return nil
	}
	return values
}
