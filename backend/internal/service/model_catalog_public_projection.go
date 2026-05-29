package service

import (
	"context"
	"sort"
	"strings"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	infraerrors "github.com/Wei-Shaw/sub2api/internal/pkg/errors"
)

func (s *ModelCatalogService) buildPublicModelCatalogItemsFromProjection(
	ctx context.Context,
	records map[string]*modelCatalogRecord,
	pricingSnapshot *BillingPricingCatalogSnapshot,
	rules []BillingRule,
) ([]PublicModelCatalogItem, error) {
	projections, err := s.publicModelCatalogProjectionEntries(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]PublicModelCatalogItem, 0, len(projections))
	for _, projection := range projections {
		item, ok := buildPublicModelCatalogItemFromProjection(projection, records, pricingSnapshot, rules)
		if !ok {
			continue
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(firstNonEmptyTrimmed(items[i].DisplayName, items[i].Model)))
		right := strings.ToLower(strings.TrimSpace(firstNonEmptyTrimmed(items[j].DisplayName, items[j].Model)))
		if left == right {
			return items[i].Model < items[j].Model
		}
		return left < right
	})
	return items, nil
}

func (s *ModelCatalogService) publicModelCatalogProjectionEntries(ctx context.Context) ([]PublicModelProjectionEntry, error) {
	if s == nil || s.gatewayService == nil {
		return nil, nil
	}
	return s.gatewayService.ListActivePublicModelProjection(ctx)
}

func (s *ModelCatalogService) findPublicModelCatalogProjection(
	ctx context.Context,
	model string,
) (PublicModelProjectionEntry, bool, error) {
	normalizedModel := NormalizeModelCatalogModelID(model)
	if normalizedModel == "" {
		return PublicModelProjectionEntry{}, false, nil
	}
	projections, err := s.publicModelCatalogProjectionEntries(ctx)
	if err != nil {
		return PublicModelProjectionEntry{}, false, err
	}
	for _, projection := range projections {
		if NormalizeModelCatalogModelID(projection.PublicID) == normalizedModel {
			return projection, true, nil
		}
		for _, candidate := range projection.SourceIDs {
			if NormalizeModelCatalogModelID(candidate) == normalizedModel {
				return projection, true, nil
			}
		}
	}
	return PublicModelProjectionEntry{}, false, nil
}

func (s *ModelCatalogService) PublicModelCatalogDetail(ctx context.Context, model string) (*PublicModelCatalogDetail, error) {
	normalizedModel := NormalizeModelCatalogModelID(model)
	if normalizedModel == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	if rawPublished := s.loadPublishedPublicModelCatalogSnapshot(ctx); rawPublished != nil {
		published := s.filterPublishedPublicModelCatalogSnapshot(ctx, rawPublished)
		if detail, ok := published.Details[normalizedModel]; ok {
			cloned := clonePublicModelCatalogDetail(detail)
			cloned.Item = sanitizePublicModelCatalogItemForPublic(cloned.Item)
			cloned.CatalogSource = PublicModelCatalogSourcePublished
			return &cloned, nil
		}
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}

	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}

	var (
		pricingSnapshot *BillingPricingCatalogSnapshot
		rules           []BillingRule
	)
	if s.billingCenterService != nil {
		pricingSnapshot, err = s.billingCenterService.ensureBillingPricingCatalogMigrated(ctx)
		if err != nil {
			return nil, err
		}
		rules = s.billingCenterService.ListRules(ctx)
	}

	var (
		item       PublicModelCatalogItem
		projection PublicModelProjectionEntry
		found      bool
	)
	if s != nil && s.gatewayService != nil {
		projection, found, err = s.findPublicModelCatalogProjection(ctx, normalizedModel)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
		}
		item, found = buildPublicModelCatalogItemFromProjection(projection, records, pricingSnapshot, rules)
		if !found {
			return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
		}
	} else {
		record, ok := resolveModelCatalogRecord(records, normalizedModel)
		if !ok || record == nil {
			return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
		}
		entry := modelregistryEntryFromRecord(record)
		item, found = buildPublicModelCatalogItem(entry, records, pricingSnapshot, rules)
		if !found {
			return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
		}
		projection = PublicModelProjectionEntry{
			PublicID:          item.Model,
			DisplayName:       item.DisplayName,
			Platform:          item.Provider,
			AvailabilityState: item.AvailabilityState,
			StaleState:        item.StaleState,
			LifecycleStatus:   item.LifecycleStatus,
			SourceIDs:         []string{record.model},
		}
	}

	exampleSource, exampleProtocol, examplePageID, exampleMarkdown, exampleOverrideID := s.buildPublicModelCatalogDetailExample(ctx, item)
	if len(projection.SourceIDs) > 0 {
		item.SourceIDs = append([]string(nil), projection.SourceIDs...)
	}
	if !s.publicModelCatalogItemRouteConfirmed(ctx, item) {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	return &PublicModelCatalogDetail{
		Item:              sanitizePublicModelCatalogItemForPublic(item),
		CatalogSource:     PublicModelCatalogSourceLiveFallback,
		ExampleSource:     exampleSource,
		ExampleProtocol:   exampleProtocol,
		ExamplePageID:     examplePageID,
		ExampleMarkdown:   exampleMarkdown,
		ExampleOverrideID: exampleOverrideID,
	}, nil
}

func (s *ModelCatalogService) PublishedPublicModelCatalogDetail(ctx context.Context, model string) (*PublicModelCatalogDetail, error) {
	normalizedModel := NormalizeModelCatalogModelID(model)
	if normalizedModel == "" {
		return nil, infraerrors.BadRequest("MODEL_REQUIRED", "model is required")
	}
	rawPublished := s.loadPublishedPublicModelCatalogSnapshot(ctx)
	if rawPublished == nil {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	published := s.filterPublishedPublicModelCatalogSnapshot(ctx, rawPublished)
	detail, ok := published.Details[normalizedModel]
	if !ok {
		return nil, infraerrors.NotFound("MODEL_NOT_FOUND", "model not found")
	}
	cloned := clonePublicModelCatalogDetail(detail)
	cloned.Item = sanitizePublicModelCatalogItemForPublic(cloned.Item)
	return &cloned, nil
}

func buildPublicModelCatalogItemFromProjection(
	projection PublicModelProjectionEntry,
	records map[string]*modelCatalogRecord,
	pricingSnapshot *BillingPricingCatalogSnapshot,
	rules []BillingRule,
) (PublicModelCatalogItem, bool) {
	record, persisted, ok := resolvePublicModelCatalogProjectionPricing(projection, records, pricingSnapshot, rules)
	if !ok {
		return PublicModelCatalogItem{}, false
	}

	metadata := billingPricingMetadataForPersistedModel(persisted)
	officialDisplay := publicModelCatalogPriceDisplayFromForm(metadata, persisted.OfficialForm)
	effectiveSaleForm := billingPricingEffectiveSaleDisplayForm(persisted)
	priceDisplay := publicModelCatalogPriceDisplayFromForm(metadata, effectiveSaleForm)
	if len(priceDisplay.Primary) == 0 && len(priceDisplay.Secondary) == 0 {
		return PublicModelCatalogItem{}, false
	}

	provider := resolvePublicModelCatalogProjectionProvider(projection, record, persisted)
	modelID := NormalizeModelCatalogModelID(firstRegistryString(projection.PublicID, persisted.Model))
	displayName := strings.TrimSpace(firstRegistryString(projection.DisplayName, persisted.DisplayName))
	if displayName == "" {
		displayName = FormatModelCatalogDisplayName(modelID)
	}

	mode := strings.TrimSpace(strings.ToLower(firstRegistryString(persisted.Mode)))
	if mode == "" && record != nil {
		mode = strings.TrimSpace(strings.ToLower(record.mode))
	}
	if mode == "" {
		mode = inferModelMode(firstNonEmptyTrimmed(firstRegistryString(projection.SourceIDs...), modelID), "")
	}
	lifecycleStatus := normalizePublicModelLifecycleStatus(
		projection.LifecycleStatus,
		projection.DisplayName,
		projection.PublicID,
		firstRegistryString(projection.SourceIDs...),
	)
	modelMetadata := publicModelCatalogMetadataForCandidates(
		records,
		record,
		projection.PublicID,
		firstRegistryString(projection.SourceIDs...),
		persisted.Model,
	)

	return PublicModelCatalogItem{
		EntryID:              publicModelCatalogEntryID(0, projection.Platform, firstNonEmptyTrimmed(firstRegistryString(projection.SourceIDs...), modelID)),
		PublicModelID:        modelID,
		Model:                modelID,
		BaseModel:            firstNonEmptyTrimmed(firstRegistryString(projection.SourceIDs...), modelID),
		SourceModelID:        firstNonEmptyTrimmed(firstRegistryString(projection.SourceIDs...), modelID),
		SourceProtocol:       strings.TrimSpace(projection.Platform),
		DisplayName:          displayName,
		Provider:             provider,
		ProviderIconKey:      provider,
		Status:               publicModelStatusFromProjection(projection.AvailabilityState, projection.StaleState, lifecycleStatus),
		AvailabilityState:    firstNonEmptyTrimmed(projection.AvailabilityState, AccountModelAvailabilityUnknown),
		StaleState:           firstNonEmptyTrimmed(projection.StaleState, AccountModelStaleStateUnverified),
		LifecycleStatus:      lifecycleStatus,
		ContextWindowTokens:  modelMetadata.ContextWindowTokens,
		Modalities:           append([]string(nil), modelMetadata.Modalities...),
		Capabilities:         append([]string(nil), modelMetadata.Capabilities...),
		RequestProtocols:     publicModelCatalogRequestProtocolsForProjection(projection, records, provider),
		SourceIDs:            append([]string(nil), projection.SourceIDs...),
		Mode:                 mode,
		Currency:             defaultModelPricingCurrency(persisted.Currency),
		PriceDisplay:         priceDisplay,
		OfficialPriceDisplay: officialDisplay,
		SalePriceDisplay:     priceDisplay,
		MultiplierSummary:    publicModelCatalogMultiplierSummaryFromForm(persisted.SaleForm),
		RuntimePriceSpec:     publicModelCatalogRuntimePriceSpecFromPersisted(persisted),
	}, true
}

func resolvePublicModelCatalogProjectionProvider(
	projection PublicModelProjectionEntry,
	record *modelCatalogRecord,
	persisted BillingPricingPersistedModel,
) string {
	if provider := NormalizeModelProvider(strings.TrimSpace(persisted.Provider)); provider != "" {
		return provider
	}
	if record != nil {
		if provider := NormalizeModelProvider(record.provider); provider != "" {
			return provider
		}
	}

	for _, candidate := range []string{
		persisted.Model,
		projection.PublicID,
		firstRegistryString(projection.SourceIDs...),
		func() string {
			if record == nil {
				return ""
			}
			return firstRegistryString(record.model, record.canonicalModelID, record.pricingLookupModelID)
		}(),
	} {
		normalized := NormalizeModelCatalogModelID(candidate)
		if normalized == "" {
			continue
		}
		if provider := NormalizeModelProvider(inferModelProvider(normalized)); provider != "" {
			return provider
		}
	}

	return NormalizeModelProvider(projection.Platform)
}

func resolvePublicModelCatalogProjectionPricing(
	projection PublicModelProjectionEntry,
	records map[string]*modelCatalogRecord,
	pricingSnapshot *BillingPricingCatalogSnapshot,
	rules []BillingRule,
) (*modelCatalogRecord, BillingPricingPersistedModel, bool) {
	for _, candidate := range publicModelCatalogProjectionLookupCandidates(projection) {
		record, hasRecord := resolveModelCatalogRecord(records, candidate)
		persisted, ok, _ := billingPricingSnapshotModel(pricingSnapshot, candidate)
		if !ok && hasRecord && record != nil {
			persisted = billingPricingPersistedModelFromRecord(record, rules)
			ok = true
		}
		if ok {
			return record, persisted, true
		}
	}
	return nil, BillingPricingPersistedModel{}, false
}

func publicModelCatalogProjectionLookupCandidates(projection PublicModelProjectionEntry) []string {
	candidates := make([]string, 0, 1+len(projection.SourceIDs)+len(projection.AliasIDs))
	appendCandidate := func(value string) {
		normalized := NormalizeModelCatalogModelID(value)
		if normalized == "" {
			return
		}
		for _, existing := range candidates {
			if existing == normalized {
				return
			}
		}
		candidates = append(candidates, normalized)
	}
	appendCandidate(projection.PublicID)
	for _, sourceID := range projection.SourceIDs {
		appendCandidate(sourceID)
	}
	for _, aliasID := range projection.AliasIDs {
		appendCandidate(aliasID)
	}
	return candidates
}

func publicModelCatalogRequestProtocolsForProjection(
	projection PublicModelProjectionEntry,
	records map[string]*modelCatalogRecord,
	provider string,
) []string {
	seen := map[string]struct{}{}
	items := make([]string, 0, 6)
	appendProtocol := func(value string) {
		protocol := publicModelCatalogProtocolFamily(value)
		if protocol == "" {
			return
		}
		if _, ok := seen[protocol]; ok {
			return
		}
		seen[protocol] = struct{}{}
		items = append(items, protocol)
	}

	appendProtocol(provider)
	appendProtocol(projection.Platform)
	appendProtocol(projection.PublicID)
	for _, sourceID := range projection.SourceIDs {
		appendProtocol(sourceID)
		if record, ok := resolveModelCatalogRecord(records, sourceID); ok && record != nil {
			for _, value := range record.defaultPlatforms {
				appendProtocol(value)
			}
			appendProtocol(record.canonicalModelID)
			appendProtocol(record.pricingLookupModelID)
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		leftOrder, leftOK := publicModelCatalogProtocolOrder[items[i]]
		rightOrder, rightOK := publicModelCatalogProtocolOrder[items[j]]
		switch {
		case leftOK && rightOK && leftOrder != rightOrder:
			return leftOrder < rightOrder
		case leftOK != rightOK:
			return leftOK
		default:
			return items[i] < items[j]
		}
	})
	return items
}

func modelregistryEntryFromRecord(record *modelCatalogRecord) modelregistry.ModelEntry {
	if record == nil {
		return modelregistry.ModelEntry{}
	}
	return modelregistry.ModelEntry{
		ID:          record.model,
		DisplayName: record.displayName,
		Provider:    record.provider,
		Platforms:   compactStrings(record.defaultPlatforms),
		ProtocolIDs: compactStrings([]string{record.model, record.pricingLookupModelID, record.canonicalModelID}),
	}
}

type publicModelCatalogMetadata struct {
	ContextWindowTokens int64
	Modalities          []string
	Capabilities        []string
}

func publicModelCatalogMetadataForCandidates(
	records map[string]*modelCatalogRecord,
	record *modelCatalogRecord,
	candidates ...string,
) publicModelCatalogMetadata {
	values := make([]string, 0, len(candidates)+3)
	values = append(values, candidates...)
	if record != nil {
		values = append(values, record.model, record.pricingLookupModelID, record.canonicalModelID)
	}
	for _, candidate := range values {
		if entry, ok := modelregistry.SeedModelByID(candidate); ok {
			return publicModelCatalogMetadataFromEntry(entry)
		}
		if records != nil {
			if candidateRecord, ok := resolveModelCatalogRecord(records, candidate); ok && candidateRecord != nil {
				entry := modelregistryEntryFromRecord(candidateRecord)
				if seed, found := modelregistry.SeedModelByID(entry.ID); found {
					return publicModelCatalogMetadataFromEntry(seed)
				}
				return publicModelCatalogMetadataFromRecord(candidateRecord)
			}
		}
	}
	if record != nil {
		return publicModelCatalogMetadataFromRecord(record)
	}
	return publicModelCatalogMetadata{}
}

func publicModelCatalogMetadataFromEntry(entry modelregistry.ModelEntry) publicModelCatalogMetadata {
	return publicModelCatalogMetadata{
		ContextWindowTokens: entry.ContextWindowTokens,
		Modalities:          append([]string(nil), entry.Modalities...),
		Capabilities:        append([]string(nil), entry.Capabilities...),
	}
}

func publicModelCatalogMetadataFromRecord(record *modelCatalogRecord) publicModelCatalogMetadata {
	if record == nil {
		return publicModelCatalogMetadata{}
	}
	mode := inferModelMode(record.model, record.mode)
	metadata := publicModelCatalogMetadata{
		Modalities:   defaultModalitiesForMode(mode),
		Capabilities: defaultCapabilitiesForMode(mode),
	}
	if tokens, ok := modelregistry.ResolveContextWindowTokens(record.pricingLookupModelID, record.canonicalModelID, record.model); ok {
		metadata.ContextWindowTokens = tokens
	}
	return metadata
}
