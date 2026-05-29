package service

import "context"

func (s *ModelCatalogService) publicModelCatalogRateLimitSummaries(
	ctx context.Context,
	items []PublicModelCatalogItem,
) map[string]*PublicModelCatalogRateLimitSummary {
	out := make(map[string]*PublicModelCatalogRateLimitSummary)
	if s == nil || s.gatewayService == nil || s.gatewayService.accountRepo == nil || len(items) == 0 {
		return out
	}
	accountCache := make(map[int64]*Account)
	for _, item := range items {
		modelID := NormalizeModelCatalogModelID(firstNonEmptyTrimmed(item.PublicModelID, item.Model))
		if modelID == "" || item.SourceAccountID <= 0 {
			continue
		}
		account, ok := accountCache[item.SourceAccountID]
		if !ok {
			fetched, err := s.gatewayService.accountRepo.GetByID(ctx, item.SourceAccountID)
			if err != nil || fetched == nil || !fetched.IsSchedulable() {
				accountCache[item.SourceAccountID] = nil
				continue
			}
			accountCache[item.SourceAccountID] = fetched
			account = fetched
		}
		summary := s.publicModelCatalogRateLimitFromAccount(ctx, account, item)
		if !publicModelCatalogRateLimitSummaryHasValue(summary) {
			continue
		}
		current := out[modelID]
		if current == nil {
			current = &PublicModelCatalogRateLimitSummary{}
			out[modelID] = current
		}
		mergePublicModelCatalogRateLimitSummary(current, summary)
	}
	for modelID, summary := range out {
		if !publicModelCatalogRateLimitSummaryHasValue(summary) {
			delete(out, modelID)
		}
	}
	return out
}

func (s *ModelCatalogService) publicModelCatalogRateLimitFromAccount(
	ctx context.Context,
	account *Account,
	item PublicModelCatalogItem,
) *PublicModelCatalogRateLimitSummary {
	if account == nil {
		return nil
	}
	summary := &PublicModelCatalogRateLimitSummary{}
	if account.IsAnthropicOAuthOrSetupToken() {
		if rpm := account.GetBaseRPM(); rpm > 0 {
			value := int64(rpm)
			summary.RPM = &value
		}
	}
	if s != nil && s.gatewayService != nil && s.gatewayService.rateLimitService != nil {
		geminiQuotaService := s.gatewayService.rateLimitService.geminiQuotaService
		if geminiQuotaService != nil {
			quota, ok := geminiQuotaService.QuotaForAccount(ctx, account)
			if !ok {
				return summary
			}
			modelClass := geminiModelClassFromName(firstNonEmptyTrimmed(item.SourceModelID, item.BaseModel, item.Model))
			if rpm := geminiMinuteLimit(quota, modelClass); rpm > 0 {
				value := rpm
				summary.RPM = &value
			}
			if rpd := geminiDailyLimit(quota, modelClass); rpd > 0 {
				value := rpd
				summary.RPD = &value
			}
		}
	}
	if !publicModelCatalogRateLimitSummaryHasValue(summary) {
		return nil
	}
	return summary
}

func mergePublicModelCatalogRateLimitSummary(
	target *PublicModelCatalogRateLimitSummary,
	next *PublicModelCatalogRateLimitSummary,
) {
	if target == nil || next == nil {
		return
	}
	target.RPM = minPositiveInt64Ptr(target.RPM, next.RPM)
	target.TPM = minPositiveInt64Ptr(target.TPM, next.TPM)
	target.RPD = minPositiveInt64Ptr(target.RPD, next.RPD)
}

func minPositiveInt64Ptr(current *int64, next *int64) *int64 {
	if next == nil || *next <= 0 {
		return current
	}
	if current == nil || *current <= 0 || *next < *current {
		value := *next
		return &value
	}
	return current
}

func publicModelCatalogRateLimitSummaryHasValue(summary *PublicModelCatalogRateLimitSummary) bool {
	return summary != nil && (positiveInt64Ptr(summary.RPM) || positiveInt64Ptr(summary.TPM) || positiveInt64Ptr(summary.RPD))
}

func positiveInt64Ptr(value *int64) bool {
	return value != nil && *value > 0
}

func clonePublicModelCatalogRateLimitSummary(summary *PublicModelCatalogRateLimitSummary) *PublicModelCatalogRateLimitSummary {
	if summary == nil {
		return nil
	}
	cloned := &PublicModelCatalogRateLimitSummary{}
	if summary.RPM != nil {
		value := *summary.RPM
		cloned.RPM = &value
	}
	if summary.TPM != nil {
		value := *summary.TPM
		cloned.TPM = &value
	}
	if summary.RPD != nil {
		value := *summary.RPD
		cloned.RPD = &value
	}
	if !publicModelCatalogRateLimitSummaryHasValue(cloned) {
		return nil
	}
	return cloned
}
