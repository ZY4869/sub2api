package service

import (
	"context"
	"strings"
)

func publicCatalogEntryIDFromContext(ctx context.Context) string {
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		return entry.EntryID
	}
	if value, ok := PublicCatalogEntryIDMetadataFromContext(ctx); ok {
		return value
	}
	return ""
}

func publicCatalogSalePriceDisplayFromContext(ctx context.Context) PublicModelCatalogPriceDisplay {
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		return clonePublicModelCatalogPriceDisplay(entry.SalePriceDisplay)
	}
	return PublicModelCatalogPriceDisplay{}
}

func publicCatalogPublicModelIDFromContext(ctx context.Context) string {
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		return entry.PublicModelID
	}
	return ""
}

func publicCatalogSourceAccountIDFromContext(ctx context.Context) int64 {
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		return entry.SourceAccountID
	}
	return 0
}

func publicCatalogCurrencyFromContext(ctx context.Context) string {
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		return defaultModelPricingCurrency(firstNonEmptyTrimmed(entry.Currency, entry.Item.Currency))
	}
	return ""
}

func publicCatalogRuntimePriceSpecFromContext(ctx context.Context) PublicModelCatalogRuntimePriceSpec {
	if entry, ok := PublishedPublicCatalogEntryFromContext(ctx); ok && entry != nil {
		spec := normalizePublicModelCatalogRuntimePriceSpec(entry.RuntimePriceSpec)
		if strings.TrimSpace(spec.Currency) == "" {
			spec.Currency = defaultModelPricingCurrency(firstNonEmptyTrimmed(entry.Currency, entry.Item.Currency))
		}
		return spec
	}
	return PublicModelCatalogRuntimePriceSpec{}
}
