package service

import (
	"fmt"
	"sort"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

func normalizeBillingPricingStatus(status BillingPricingStatus) BillingPricingStatus {
	switch status {
	case BillingPricingStatusOK, BillingPricingStatusFallback, BillingPricingStatusConflict, BillingPricingStatusMissing:
		return status
	default:
		return BillingPricingStatusMissing
	}
}

func applyBillingPricingStatus(details []modelregistry.AdminModelDetail, records map[string]*modelCatalogRecord) {
	if len(records) == 0 {
		return
	}
	conflictWarnings, pricingLookupWarnings := billingPricingCollisionWarningsByModel(details)
	for _, record := range records {
		if record == nil {
			continue
		}
		key := NormalizeModelCatalogModelID(record.model)
		status, warnings := deriveBillingPricingStatus(
			record,
			conflictWarnings[key],
			pricingLookupWarnings[key],
		)
		record.pricingStatus = status
		record.pricingWarnings = warnings
	}
}

func billingPricingCollisionWarningsByModel(details []modelregistry.AdminModelDetail) (map[string][]string, map[string][]string) {
	collisions := collectBillingIdentifierCollisions(details)
	conflictWarnings := make(map[string][]string, len(collisions))
	pricingLookupWarnings := make(map[string][]string, len(collisions))
	for _, collision := range collisions {
		message := fmt.Sprintf("%s identifier %q collides with %d models", collision.Source, collision.Identifier, collision.Count)
		target := conflictWarnings
		if collision.Source == "pricing_lookup_ids" {
			message = fmt.Sprintf(
				"Shared pricing lookup %q is reused by %d models; pricing is sourced from the same upstream entry.",
				collision.Identifier,
				collision.Count,
			)
			target = pricingLookupWarnings
		}
		for _, modelID := range collision.Models {
			key := NormalizeModelCatalogModelID(modelID)
			target[key] = append(target[key], message)
		}
	}
	for key, items := range conflictWarnings {
		conflictWarnings[key] = compactStrings(items)
	}
	for key, items := range pricingLookupWarnings {
		pricingLookupWarnings[key] = compactStrings(items)
	}
	return conflictWarnings, pricingLookupWarnings
}

func deriveBillingPricingStatus(
	record *modelCatalogRecord,
	conflictWarnings []string,
	pricingLookupWarnings []string,
) (BillingPricingStatus, []string) {
	status := BillingPricingStatusMissing
	warnings := append([]string(nil), conflictWarnings...)
	warnings = append(warnings, pricingLookupWarnings...)

	switch {
	case record != nil && record.basePricingSource == ModelCatalogPricingSourceDynamic && !pricingEmpty(record.upstreamPricing):
		status = BillingPricingStatusOK
	case record != nil && record.basePricingSource == ModelCatalogPricingSourceFallback && !pricingEmpty(record.upstreamPricing):
		status = BillingPricingStatusFallback
		warnings = append(warnings, "Using billing fallback pricing source.")
	default:
		warnings = append(warnings, "No stable upstream pricing source found.")
		if record != nil && (record.officialOverridePricing != nil || record.saleOverridePricing != nil) {
			warnings = append(warnings, "Manual override pricing exists without an upstream source.")
		}
	}

	if len(conflictWarnings) > 0 {
		status = BillingPricingStatusConflict
	}
	return status, compactStrings(warnings)
}

func billingPricingStatusForRecord(record *modelCatalogRecord) BillingPricingStatus {
	if record == nil {
		return BillingPricingStatusMissing
	}
	return normalizeBillingPricingStatus(record.pricingStatus)
}

func billingPricingWarningsForRecord(record *modelCatalogRecord) []string {
	if record == nil {
		return nil
	}
	return compactStrings(record.pricingWarnings)
}

type billingPricingStatusSummary struct {
	ok       int
	fallback int
	conflict int
	missing  int
}

func summarizeBillingPricingStatuses(models []BillingPricingPersistedModel) billingPricingStatusSummary {
	summary := billingPricingStatusSummary{}
	for _, model := range models {
		switch normalizeBillingPricingStatus(model.PricingStatus) {
		case BillingPricingStatusOK:
			summary.ok++
		case BillingPricingStatusFallback:
			summary.fallback++
		case BillingPricingStatusConflict:
			summary.conflict++
		default:
			summary.missing++
		}
	}
	return summary
}

func billingPricingSnapshotNeedsStatusMigration(snapshot *BillingPricingCatalogSnapshot) bool {
	if snapshot == nil {
		return false
	}
	for _, model := range snapshot.Models {
		if model.PricingStatus == "" {
			return true
		}
	}
	return false
}

func assertUniqueBillingPricingModels(models []BillingPricingPersistedModel) []string {
	counts := map[string]int{}
	for _, model := range models {
		key := NormalizeModelCatalogModelID(model.Model)
		if key == "" {
			continue
		}
		counts[key]++
	}
	duplicates := make([]string, 0)
	for key, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, key)
		}
	}
	sort.Strings(duplicates)
	return duplicates
}
