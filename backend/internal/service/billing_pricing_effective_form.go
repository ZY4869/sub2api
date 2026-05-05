package service

func billingPricingEffectiveSaleDisplayForm(model BillingPricingPersistedModel) BillingPricingLayerForm {
	return billingPricingEffectiveDisplayForm(model.OfficialForm, model.SaleForm)
}

func billingPricingEffectiveDisplayForm(
	official BillingPricingLayerForm,
	sale BillingPricingLayerForm,
) BillingPricingLayerForm {
	official = normalizeBillingPricingLayerFormForLayer(official, BillingLayerOfficial)
	sale = normalizeBillingPricingLayerFormForLayer(sale, BillingLayerSale)

	effective := BillingPricingLayerForm{
		Special: BillingPricingSimpleSpecial{},
	}

	for _, fieldID := range billingPricingDisplayFieldIDs() {
		value := billingPricingEffectiveDisplayFieldValue(official, sale, fieldID)
		if value == nil {
			continue
		}
		billingPricingAssignDisplayFieldValue(&effective, fieldID, value)
	}

	effective.SpecialEnabled = billingPricingSimpleSpecialEnabled(effective.Special)
	effective.TierThresholdTokens = billingPricingEffectiveDisplayThreshold(official, sale)
	effective.TieredEnabled = effective.InputPriceAboveThreshold != nil || effective.OutputPriceAboveThreshold != nil
	if !effective.TieredEnabled {
		effective.TierThresholdTokens = nil
	}
	return effective
}

func billingPricingDisplayFieldIDs() []string {
	return []string{
		billingDiscountFieldInputPrice,
		billingDiscountFieldOutputPrice,
		billingDiscountFieldCachePrice,
		billingDiscountFieldInputPriceAboveThreshold,
		billingDiscountFieldOutputPriceAboveThreshold,
		billingDiscountFieldBatchInputPrice,
		billingDiscountFieldBatchOutputPrice,
		billingDiscountFieldBatchCachePrice,
		billingDiscountFieldGroundingSearch,
		billingDiscountFieldGroundingMaps,
		billingDiscountFieldFileSearchEmbedding,
		billingDiscountFieldFileSearchRetrieval,
	}
}

func billingPricingEffectiveDisplayFieldValue(
	official BillingPricingLayerForm,
	sale BillingPricingLayerForm,
	fieldID string,
) *float64 {
	if billingPricingFieldConfigured(sale, fieldID) {
		return billingPricingEffectiveFieldValue(sale, fieldID)
	}
	if billingPricingFieldConfigured(official, fieldID) {
		return billingPricingEffectiveFieldValue(official, fieldID)
	}
	return nil
}

func billingPricingFieldConfigured(form BillingPricingLayerForm, fieldID string) bool {
	values := billingPricingConfiguredFieldValues(form)
	value, ok := values[fieldID]
	return ok && value != nil
}

func billingPricingAssignDisplayFieldValue(form *BillingPricingLayerForm, fieldID string, value *float64) {
	if form == nil || value == nil {
		return
	}
	switch fieldID {
	case billingDiscountFieldInputPrice:
		form.InputPrice = cloneBillingFloat64(value)
	case billingDiscountFieldOutputPrice:
		form.OutputPrice = cloneBillingFloat64(value)
	case billingDiscountFieldCachePrice:
		form.CachePrice = cloneBillingFloat64(value)
	case billingDiscountFieldInputPriceAboveThreshold:
		form.InputPriceAboveThreshold = cloneBillingFloat64(value)
	case billingDiscountFieldOutputPriceAboveThreshold:
		form.OutputPriceAboveThreshold = cloneBillingFloat64(value)
	case billingDiscountFieldBatchInputPrice:
		form.Special.BatchInputPrice = cloneBillingFloat64(value)
	case billingDiscountFieldBatchOutputPrice:
		form.Special.BatchOutputPrice = cloneBillingFloat64(value)
	case billingDiscountFieldBatchCachePrice:
		form.Special.BatchCachePrice = cloneBillingFloat64(value)
	case billingDiscountFieldGroundingSearch:
		form.Special.GroundingSearch = cloneBillingFloat64(value)
	case billingDiscountFieldGroundingMaps:
		form.Special.GroundingMaps = cloneBillingFloat64(value)
	case billingDiscountFieldFileSearchEmbedding:
		form.Special.FileSearchEmbedding = cloneBillingFloat64(value)
	case billingDiscountFieldFileSearchRetrieval:
		form.Special.FileSearchRetrieval = cloneBillingFloat64(value)
	}
}

func billingPricingEffectiveDisplayThreshold(
	official BillingPricingLayerForm,
	sale BillingPricingLayerForm,
) *int {
	if sale.TieredEnabled && sale.TierThresholdTokens != nil {
		return cloneBillingInt(sale.TierThresholdTokens)
	}
	if official.TieredEnabled && official.TierThresholdTokens != nil {
		return cloneBillingInt(official.TierThresholdTokens)
	}
	return nil
}
