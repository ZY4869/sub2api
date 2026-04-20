package service

import (
	"context"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
)

type BillingPricingAudit struct {
	TotalModels             int                                 `json:"total_models"`
	DuplicateModelIDs       []string                            `json:"duplicate_model_ids"`
	AuxIdentifierCollisions []BillingPricingIdentifierCollision `json:"aux_identifier_collisions"`
	MissingInSnapshotCount  int                                 `json:"missing_in_snapshot_count"`
	MissingInSnapshotModels []string                            `json:"missing_in_snapshot_models"`
	SnapshotOnlyCount       int                                 `json:"snapshot_only_count"`
	RefreshRequired         bool                                `json:"refresh_required"`
	SnapshotUpdatedAt       *time.Time                          `json:"snapshot_updated_at,omitempty"`
}

type BillingPricingIdentifierCollision struct {
	Source     string   `json:"source"`
	Identifier string   `json:"identifier"`
	Models     []string `json:"models"`
	Count      int      `json:"count"`
}

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

	audit := &BillingPricingAudit{
		TotalModels:             len(records),
		DuplicateModelIDs:       collectBillingDuplicateModelIDs(details),
		AuxIdentifierCollisions: collectBillingIdentifierCollisions(details),
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
		snapshotOnlyCount++
	}
	audit.SnapshotOnlyCount = snapshotOnlyCount
	audit.RefreshRequired = audit.MissingInSnapshotCount > 0 || audit.SnapshotOnlyCount > 0
	return audit, nil
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
