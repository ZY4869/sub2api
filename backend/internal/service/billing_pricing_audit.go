package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

type BillingPricingAudit struct {
	TotalModels             int                                   `json:"total_models"`
	PricingStatusCounts     BillingPricingStatusCounts            `json:"pricing_status_counts"`
	DuplicateModelIDs       []string                              `json:"duplicate_model_ids"`
	AuxIdentifierCollisions []BillingPricingIdentifierCollision   `json:"aux_identifier_collisions"`
	CollisionCountsBySource BillingPricingCollisionCountsBySource `json:"collision_counts_by_source"`
	ProviderIssueCounts     []BillingPricingProviderIssueCount    `json:"provider_issue_counts,omitempty"`
	PricingIssueExamples    []BillingPricingIssueExample          `json:"pricing_issue_examples,omitempty"`
	MissingInSnapshotCount  int                                   `json:"missing_in_snapshot_count"`
	MissingInSnapshotModels []string                              `json:"missing_in_snapshot_models"`
	SnapshotOnlyCount       int                                   `json:"snapshot_only_count"`
	SnapshotOnlyModels      []string                              `json:"snapshot_only_models,omitempty"`
	RefreshRequired         bool                                  `json:"refresh_required"`
	SnapshotUpdatedAt       *time.Time                            `json:"snapshot_updated_at,omitempty"`
}

type BillingPricingIdentifierCollision struct {
	Source     string   `json:"source"`
	Identifier string   `json:"identifier"`
	Models     []string `json:"models"`
	Count      int      `json:"count"`
}

type BillingPricingStatusCounts struct {
	OK       int `json:"ok"`
	Fallback int `json:"fallback"`
	Conflict int `json:"conflict"`
	Missing  int `json:"missing"`
}

type BillingPricingCollisionCountsBySource struct {
	Aliases          int `json:"aliases"`
	ProtocolIDs      int `json:"protocol_ids"`
	PricingLookupIDs int `json:"pricing_lookup_ids"`
}

type BillingPricingProviderIssueCount struct {
	Provider string `json:"provider"`
	Total    int    `json:"total"`
	Fallback int    `json:"fallback"`
	Conflict int    `json:"conflict"`
	Missing  int    `json:"missing"`
}

type BillingPricingIssueExample struct {
	Model         string               `json:"model"`
	DisplayName   string               `json:"display_name,omitempty"`
	Provider      string               `json:"provider,omitempty"`
	PricingStatus BillingPricingStatus `json:"pricing_status"`
	FirstWarning  string               `json:"first_warning,omitempty"`
}

const billingPricingAuditIssueExampleLimit = 12

func (s *BillingCenterService) GetPricingAudit(ctx context.Context) (*BillingPricingAudit, error) {
	if s == nil || s.modelCatalogService == nil {
		return &BillingPricingAudit{}, nil
	}

	details, err := s.modelCatalogService.catalogBaselineEntries(ctx)
	if err != nil {
		return nil, err
	}
	records, err := s.modelCatalogService.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}

	collisions := collectBillingIdentifierCollisions(details)
	audit := &BillingPricingAudit{
		TotalModels:             len(records),
		PricingStatusCounts:     summarizeBillingPricingStatusCounts(records),
		DuplicateModelIDs:       collectBillingDuplicateModelIDs(details),
		AuxIdentifierCollisions: collisions,
		CollisionCountsBySource: countBillingCollisionSources(collisions),
		ProviderIssueCounts:     collectBillingProviderIssueCounts(records),
		PricingIssueExamples:    collectBillingPricingIssueExamples(records, billingPricingAuditIssueExampleLimit),
	}

	currentModels := make(map[string]struct{}, len(records))
	for modelID := range records {
		normalized := NormalizeModelCatalogModelID(modelID)
		if normalized == "" {
			continue
		}
		currentModels[normalized] = struct{}{}
	}

	snapshot := s.loadBillingPricingCatalogSnapshot(ctx)
	snapshotModels := map[string]struct{}{}
	if snapshot != nil {
		if !snapshot.UpdatedAt.IsZero() {
			updatedAt := snapshot.UpdatedAt.UTC()
			audit.SnapshotUpdatedAt = &updatedAt
		}
		snapshotModels = make(map[string]struct{}, len(snapshot.Models))
		for _, model := range snapshot.Models {
			normalized := NormalizeModelCatalogModelID(model.Model)
			if normalized == "" {
				continue
			}
			snapshotModels[normalized] = struct{}{}
		}
	}

	for modelID := range currentModels {
		if _, ok := snapshotModels[modelID]; ok {
			continue
		}
		audit.MissingInSnapshotModels = append(audit.MissingInSnapshotModels, modelID)
	}
	sort.Strings(audit.MissingInSnapshotModels)
	audit.MissingInSnapshotCount = len(audit.MissingInSnapshotModels)

	snapshotOnlyCount := 0
	for modelID := range snapshotModels {
		if _, ok := currentModels[modelID]; ok {
			continue
		}
		audit.SnapshotOnlyModels = append(audit.SnapshotOnlyModels, modelID)
		snapshotOnlyCount++
	}
	sort.Strings(audit.SnapshotOnlyModels)
	audit.SnapshotOnlyCount = snapshotOnlyCount
	audit.RefreshRequired = audit.MissingInSnapshotCount > 0 || audit.SnapshotOnlyCount > 0
	return audit, nil
}

func summarizeBillingPricingStatusCounts(records map[string]*modelCatalogRecord) BillingPricingStatusCounts {
	counts := BillingPricingStatusCounts{}
	for _, record := range records {
		switch billingPricingStatusForRecord(record) {
		case BillingPricingStatusOK:
			counts.OK++
		case BillingPricingStatusFallback:
			counts.Fallback++
		case BillingPricingStatusConflict:
			counts.Conflict++
		default:
			counts.Missing++
		}
	}
	return counts
}

func countBillingCollisionSources(collisions []BillingPricingIdentifierCollision) BillingPricingCollisionCountsBySource {
	counts := BillingPricingCollisionCountsBySource{}
	for _, collision := range collisions {
		switch strings.TrimSpace(strings.ToLower(collision.Source)) {
		case "aliases":
			counts.Aliases++
		case "protocol_ids":
			counts.ProtocolIDs++
		case "pricing_lookup_ids":
			counts.PricingLookupIDs++
		}
	}
	return counts
}

func collectBillingProviderIssueCounts(records map[string]*modelCatalogRecord) []BillingPricingProviderIssueCount {
	countsByProvider := map[string]*BillingPricingProviderIssueCount{}
	for _, record := range records {
		status := billingPricingStatusForRecord(record)
		if status == BillingPricingStatusOK {
			continue
		}
		provider := NormalizeModelProvider(record.provider)
		if provider == "" {
			provider = "unknown"
		}
		counts := countsByProvider[provider]
		if counts == nil {
			counts = &BillingPricingProviderIssueCount{Provider: provider}
			countsByProvider[provider] = counts
		}
		counts.Total++
		switch status {
		case BillingPricingStatusFallback:
			counts.Fallback++
		case BillingPricingStatusConflict:
			counts.Conflict++
		default:
			counts.Missing++
		}
	}

	items := make([]BillingPricingProviderIssueCount, 0, len(countsByProvider))
	for _, item := range countsByProvider {
		if item == nil || item.Total == 0 {
			continue
		}
		items = append(items, *item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Total == items[j].Total {
			if items[i].Conflict == items[j].Conflict {
				if items[i].Missing == items[j].Missing {
					if items[i].Fallback == items[j].Fallback {
						return items[i].Provider < items[j].Provider
					}
					return items[i].Fallback > items[j].Fallback
				}
				return items[i].Missing > items[j].Missing
			}
			return items[i].Conflict > items[j].Conflict
		}
		return items[i].Total > items[j].Total
	})
	return items
}

func collectBillingPricingIssueExamples(records map[string]*modelCatalogRecord, limit int) []BillingPricingIssueExample {
	items := make([]BillingPricingIssueExample, 0)
	for _, record := range records {
		status := billingPricingStatusForRecord(record)
		if status == BillingPricingStatusOK {
			continue
		}
		warnings := billingPricingWarningsForRecord(record)
		firstWarning := ""
		if len(warnings) > 0 {
			firstWarning = warnings[0]
		}
		items = append(items, BillingPricingIssueExample{
			Model:         NormalizeModelCatalogModelID(record.model),
			DisplayName:   record.displayName,
			Provider:      NormalizeModelProvider(record.provider),
			PricingStatus: status,
			FirstWarning:  firstWarning,
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		if billingPricingIssuePriority(items[i].PricingStatus) == billingPricingIssuePriority(items[j].PricingStatus) {
			if items[i].Provider == items[j].Provider {
				if items[i].DisplayName == items[j].DisplayName {
					return items[i].Model < items[j].Model
				}
				return items[i].DisplayName < items[j].DisplayName
			}
			return items[i].Provider < items[j].Provider
		}
		return billingPricingIssuePriority(items[i].PricingStatus) < billingPricingIssuePriority(items[j].PricingStatus)
	})
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	return items
}

func billingPricingIssuePriority(status BillingPricingStatus) int {
	switch normalizeBillingPricingStatus(status) {
	case BillingPricingStatusConflict:
		return 0
	case BillingPricingStatusMissing:
		return 1
	case BillingPricingStatusFallback:
		return 2
	default:
		return 3
	}
}

func collectBillingDuplicateModelIDs(details []modelregistry.AdminModelDetail) []string {
	counts := map[string]int{}
	for _, detail := range details {
		modelID := NormalizeModelCatalogModelID(detail.ID)
		if modelID == "" {
			continue
		}
		counts[modelID]++
	}
	duplicates := make([]string, 0)
	for modelID, count := range counts {
		if count > 1 {
			duplicates = append(duplicates, modelID)
		}
	}
	sort.Strings(duplicates)
	return duplicates
}

func collectBillingIdentifierCollisions(details []modelregistry.AdminModelDetail) []BillingPricingIdentifierCollision {
	type identifierKey struct {
		source     string
		identifier string
	}

	seen := map[identifierKey]map[string]struct{}{}
	appendIdentifier := func(source string, raw string, modelID string) {
		identifier := canonicalizeBillingAuditIdentifier(raw)
		if identifier == "" || modelID == "" {
			return
		}
		key := identifierKey{source: source, identifier: identifier}
		models := seen[key]
		if models == nil {
			models = map[string]struct{}{}
			seen[key] = models
		}
		models[modelID] = struct{}{}
	}

	for _, detail := range details {
		modelID := NormalizeModelCatalogModelID(detail.ID)
		if modelID == "" {
			continue
		}
		for _, alias := range detail.Aliases {
			appendIdentifier("aliases", alias, modelID)
		}
		for _, protocolID := range detail.ProtocolIDs {
			appendIdentifier("protocol_ids", protocolID, modelID)
		}
		for _, lookupID := range detail.PricingLookupIDs {
			appendIdentifier("pricing_lookup_ids", lookupID, modelID)
		}
	}

	collisions := make([]BillingPricingIdentifierCollision, 0)
	for key, models := range seen {
		if len(models) <= 1 {
			continue
		}
		items := make([]string, 0, len(models))
		for modelID := range models {
			items = append(items, modelID)
		}
		sort.Strings(items)
		collisions = append(collisions, BillingPricingIdentifierCollision{
			Source:     key.source,
			Identifier: key.identifier,
			Models:     items,
			Count:      len(items),
		})
	}
	sort.SliceStable(collisions, func(i, j int) bool {
		if collisions[i].Source == collisions[j].Source {
			return collisions[i].Identifier < collisions[j].Identifier
		}
		return collisions[i].Source < collisions[j].Source
	})
	return collisions
}

func canonicalizeBillingAuditIdentifier(value string) string {
	canonical := CanonicalizeModelNameForPricing(value)
	if canonical != "" {
		return canonical
	}
	return strings.TrimSpace(strings.ToLower(value))
}
