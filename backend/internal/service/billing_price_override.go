package service

func (s *BillingService) ReplaceModelPriceOverrides(overrides map[string]*ModelPricingOverride) {
	normalized := make(map[string]*ModelPricingOverride, len(overrides))
	for model, override := range overrides {
		key := CanonicalizeModelNameForPricing(model)
		if key == "" || override == nil || pricingEmpty(&override.ModelCatalogPricing) {
			continue
		}
		normalized[key] = cloneModelPricingOverride(override)
	}
	s.overrideMu.Lock()
	s.priceOverrides = normalized
	s.overrideMu.Unlock()
}

func (s *BillingService) ReplaceModelOfficialPriceOverrides(overrides map[string]*ModelPricingOverride) {
	normalized := make(map[string]*ModelPricingOverride, len(overrides))
	for model, override := range overrides {
		key := CanonicalizeModelNameForPricing(model)
		if key == "" || override == nil || pricingEmpty(&override.ModelCatalogPricing) {
			continue
		}
		normalized[key] = cloneModelPricingOverride(override)
	}
	s.overrideMu.Lock()
	s.officialPriceOverrides = normalized
	s.overrideMu.Unlock()
}

func (s *BillingService) getModelOfficialPriceOverride(model string) *ModelPricingOverride {
	key := CanonicalizeModelNameForPricing(model)
	if key == "" {
		return nil
	}
	s.overrideMu.RLock()
	override := s.officialPriceOverrides[key]
	s.overrideMu.RUnlock()
	return cloneModelPricingOverride(override)
}

func (s *BillingService) getModelPriceOverride(model string) *ModelPricingOverride {
	key := CanonicalizeModelNameForPricing(model)
	if key == "" {
		return nil
	}
	s.overrideMu.RLock()
	override := s.priceOverrides[key]
	s.overrideMu.RUnlock()
	return cloneModelPricingOverride(override)
}

func applyModelPricingOverride(pricing *ModelPricing, override *ModelPricingOverride) *ModelPricing {
	if pricing == nil || override == nil {
		return pricing
	}
	cloned := *pricing
	if currency := normalizeModelPricingCurrency(override.Currency); currency != "" {
		applyModelPricingCurrencyMetadata(&cloned, pricingCurrencyMetadataFromCatalog(&override.ModelCatalogPricing))
	}
	if override.InputCostPerToken != nil {
		cloned.InputPricePerToken = *override.InputCostPerToken
	}
	if override.InputCostPerTokenPriority != nil {
		cloned.InputPricePerTokenPriority = *override.InputCostPerTokenPriority
	}
	if override.InputTokenThreshold != nil {
		cloned.InputTokenThreshold = *override.InputTokenThreshold
	}
	if override.InputCostPerTokenAboveThreshold != nil {
		cloned.InputPricePerTokenAboveThreshold = *override.InputCostPerTokenAboveThreshold
	}
	if override.InputCostPerTokenPriorityAboveThreshold != nil {
		cloned.InputPricePerTokenPriorityAboveThreshold = *override.InputCostPerTokenPriorityAboveThreshold
	}
	if override.OutputCostPerToken != nil {
		cloned.OutputPricePerToken = *override.OutputCostPerToken
	}
	if override.OutputCostPerTokenPriority != nil {
		cloned.OutputPricePerTokenPriority = *override.OutputCostPerTokenPriority
	}
	if override.OutputTokenThreshold != nil {
		cloned.OutputTokenThreshold = *override.OutputTokenThreshold
	}
	if override.OutputCostPerTokenAboveThreshold != nil {
		cloned.OutputPricePerTokenAboveThreshold = *override.OutputCostPerTokenAboveThreshold
	}
	if override.OutputCostPerTokenPriorityAboveThreshold != nil {
		cloned.OutputPricePerTokenPriorityAboveThreshold = *override.OutputCostPerTokenPriorityAboveThreshold
	}
	if override.CacheCreationInputTokenCost != nil {
		cloned.CacheCreationPricePerToken = *override.CacheCreationInputTokenCost
	}
	if override.CacheCreationInputTokenCostAbove1hr != nil {
		cloned.CacheCreation1hPrice = *override.CacheCreationInputTokenCostAbove1hr
	}
	if override.CacheReadInputTokenCost != nil {
		cloned.CacheReadPricePerToken = *override.CacheReadInputTokenCost
	}
	if override.CacheReadInputTokenCostPriority != nil {
		cloned.CacheReadPricePerTokenPriority = *override.CacheReadInputTokenCostPriority
	}
	if override.OutputCostPerImage != nil {
		cloned.OutputPricePerImage = *override.OutputCostPerImage
	}
	if override.OutputCostPerImagePriority != nil {
		cloned.OutputPricePerImagePriority = *override.OutputCostPerImagePriority
	}
	if override.OutputCostPerVideoRequest != nil {
		cloned.OutputPricePerVideoRequest = *override.OutputCostPerVideoRequest
	}
	return &cloned
}
