package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/modelregistry"
	"github.com/Wei-Shaw/sub2api/internal/pkg/logger"
	"go.uber.org/zap"
)

const (
	publicModelCatalogMultiplierDisabled = "disabled"
	publicModelCatalogMultiplierUniform  = "uniform"
	publicModelCatalogMultiplierMixed    = "mixed"
	publicModelCatalogProtocolVertex     = "vertex-batch"
)

var publicModelCatalogProtocolOrder = map[string]int{
	PlatformOpenAI:                   0,
	PlatformAnthropic:                1,
	PlatformGemini:                   2,
	PlatformGrok:                     3,
	PlatformAntigravity:              4,
	publicModelCatalogProtocolVertex: 5,
}

func (s *ModelCatalogService) PublicModelCatalogSnapshot(ctx context.Context) (*PublicModelCatalogSnapshot, error) {
	if published := s.loadPublishedPublicModelCatalogSnapshot(ctx); published != nil {
		snapshot := sanitizePublicModelCatalogSnapshotForPublic(&published.Snapshot)
		snapshot.CatalogSource = PublicModelCatalogSourcePublished
		return snapshot, nil
	}
	if snapshot, age, ok := s.getFreshPublicModelCatalogSnapshot(); ok {
		snapshot.CatalogSource = PublicModelCatalogSourceLiveFallback
		logger.FromContext(ctx).Info(
			"public model catalog live fallback cache hit",
			zap.String("component", "service.model_catalog"),
			zap.Duration("cache_age", age),
			zap.Int("model_count", len(snapshot.Items)),
		)
		return snapshot, nil
	}

	snapshot, err := s.buildLivePublicModelCatalogSnapshot(ctx)
	if err != nil {
		if fallback, age, ok := s.getAnyPublicModelCatalogSnapshot(); ok {
			fallback.CatalogSource = PublicModelCatalogSourceLiveFallback
			logger.FromContext(ctx).Warn(
				"public model catalog live fallback stale cache",
				zap.String("component", "service.model_catalog"),
				zap.Duration("cache_age", age),
				zap.Int("model_count", len(fallback.Items)),
				zap.Error(err),
			)
			return fallback, nil
		}
		return nil, err
	}

	s.storePublicModelCatalogSnapshot(snapshot)
	liveSnapshot := clonePublicModelCatalogSnapshot(snapshot)
	liveSnapshot.CatalogSource = PublicModelCatalogSourceLiveFallback
	logger.FromContext(ctx).Info(
		"public model catalog live fallback rebuilt",
		zap.String("component", "service.model_catalog"),
		zap.Int("model_count", len(liveSnapshot.Items)),
	)
	return liveSnapshot, nil
}

func emptyPublishedPublicModelCatalogSnapshot() *PublicModelCatalogSnapshot {
	return &PublicModelCatalogSnapshot{
		PageSize: normalizePublicModelCatalogPageSize(defaultPublicModelCatalogPageSize),
		Items:    []PublicModelCatalogItem{},
	}
}

func (s *ModelCatalogService) PublishedPublicModelCatalogSnapshot(ctx context.Context) (*PublicModelCatalogSnapshot, error) {
	if published := s.loadPublishedPublicModelCatalogSnapshot(ctx); published != nil {
		return sanitizePublicModelCatalogSnapshotForPublic(&published.Snapshot), nil
	}
	return emptyPublishedPublicModelCatalogSnapshot(), nil
}

func (s *ModelCatalogService) buildLivePublicModelCatalogSnapshot(ctx context.Context) (*PublicModelCatalogSnapshot, error) {
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

	var items []PublicModelCatalogItem
	providerBuckets := map[string]struct{}{}
	protocolBuckets := map[string]struct{}{}
	multiplierBuckets := map[string]struct{}{}
	if s != nil && s.gatewayService != nil {
		var built bool
		items, built, err = s.buildPublicModelCatalogAccountEntryItems(ctx, records, pricingSnapshot, rules)
		if err != nil {
			return nil, err
		}
		if !built {
			items, err = s.buildPublicModelCatalogItemsFromProjection(ctx, records, pricingSnapshot, rules)
			if err != nil {
				return nil, err
			}
		}
	} else {
		visibleModels, err := s.publicModelCatalogVisibleEntries(ctx)
		if err != nil {
			return nil, err
		}
		items = make([]PublicModelCatalogItem, 0, len(visibleModels))
		for _, entry := range visibleModels {
			item, ok := buildPublicModelCatalogItem(entry, records, pricingSnapshot, rules)
			if !ok {
				continue
			}
			items = append(items, item)
		}
	}
	for _, item := range items {
		if item.Provider != "" {
			providerBuckets[item.Provider] = struct{}{}
		}
		for _, protocol := range item.RequestProtocols {
			protocolBuckets[protocol] = struct{}{}
		}
		multiplierBuckets[item.MultiplierSummary.Kind] = struct{}{}
	}

	pageSize := normalizePublicModelCatalogPageSize(defaultPublicModelCatalogPageSize)
	etag, err := computePublicModelCatalogETagWithPageSize(pageSize, items)
	if err != nil {
		return nil, err
	}

	updatedAt := time.Now().UTC()
	if pricingSnapshot != nil && !pricingSnapshot.UpdatedAt.IsZero() {
		updatedAt = pricingSnapshot.UpdatedAt.UTC()
	}

	logger.FromContext(ctx).Info(
		"public model catalog snapshot cache rebuild",
		zap.String("component", "service.model_catalog"),
		zap.Int("model_count", len(items)),
		zap.Int("provider_count", len(providerBuckets)),
		zap.Int("protocol_bucket_count", len(protocolBuckets)),
		zap.Int("multiplier_bucket_count", len(multiplierBuckets)),
	)

	return &PublicModelCatalogSnapshot{
		ETag:      etag,
		UpdatedAt: updatedAt.Format(time.RFC3339),
		PageSize:  pageSize,
		Items:     items,
	}, nil
}

func (s *ModelCatalogService) publicModelCatalogTTL() time.Duration {
	if s == nil || s.publicCatalogTTL <= 0 {
		return 60 * time.Second
	}
	return s.publicCatalogTTL
}

func (s *ModelCatalogService) getFreshPublicModelCatalogSnapshot() (*PublicModelCatalogSnapshot, time.Duration, bool) {
	return s.getFreshPublicModelCatalogSnapshotWithTTL(s.publicModelCatalogTTL())
}

func (s *ModelCatalogService) getFreshPublicModelCatalogSnapshotWithTTL(ttl time.Duration) (*PublicModelCatalogSnapshot, time.Duration, bool) {
	if s == nil {
		return nil, 0, false
	}
	if ttl <= 0 {
		ttl = s.publicModelCatalogTTL()
	}
	s.publicCatalogCacheMu.RLock()
	defer s.publicCatalogCacheMu.RUnlock()
	if s.publicCatalogCache == nil || s.publicCatalogBuiltAt.IsZero() {
		return nil, 0, false
	}
	age := time.Since(s.publicCatalogBuiltAt)
	if age > ttl {
		return nil, age, false
	}
	return clonePublicModelCatalogSnapshot(s.publicCatalogCache), age, true
}

func (s *ModelCatalogService) getAnyPublicModelCatalogSnapshot() (*PublicModelCatalogSnapshot, time.Duration, bool) {
	if s == nil {
		return nil, 0, false
	}
	s.publicCatalogCacheMu.RLock()
	defer s.publicCatalogCacheMu.RUnlock()
	if s.publicCatalogCache == nil || s.publicCatalogBuiltAt.IsZero() {
		return nil, 0, false
	}
	return clonePublicModelCatalogSnapshot(s.publicCatalogCache), time.Since(s.publicCatalogBuiltAt), true
}

func (s *ModelCatalogService) storePublicModelCatalogSnapshot(snapshot *PublicModelCatalogSnapshot) {
	if s == nil || snapshot == nil {
		return
	}
	s.publicCatalogCacheMu.Lock()
	defer s.publicCatalogCacheMu.Unlock()
	s.publicCatalogCache = clonePublicModelCatalogSnapshot(snapshot)
	s.publicCatalogBuiltAt = time.Now().UTC()
}

func (s *ModelCatalogService) publicModelCatalogVisibleEntries(ctx context.Context) ([]modelregistry.ModelEntry, error) {
	if s != nil && s.modelRegistryService != nil {
		models, _, err := s.modelRegistryService.visibleSnapshotData(ctx)
		if err != nil {
			return nil, err
		}
		return models, nil
	}

	records, err := s.buildCatalogRecords(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]modelregistry.ModelEntry, 0, len(records))
	for _, record := range records {
		if record == nil {
			continue
		}
		items = append(items, modelregistry.ModelEntry{
			ID:          record.model,
			DisplayName: record.displayName,
			Provider:    record.provider,
			Platforms:   compactStrings(record.defaultPlatforms),
			ProtocolIDs: compactStrings([]string{record.model, record.pricingLookupModelID, record.canonicalModelID}),
		})
	}
	sort.SliceStable(items, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(items[i].DisplayName))
		right := strings.ToLower(strings.TrimSpace(items[j].DisplayName))
		if left == right {
			return items[i].ID < items[j].ID
		}
		return left < right
	})
	return items, nil
}

func buildPublicModelCatalogItem(
	entry modelregistry.ModelEntry,
	records map[string]*modelCatalogRecord,
	pricingSnapshot *BillingPricingCatalogSnapshot,
	rules []BillingRule,
) (PublicModelCatalogItem, bool) {
	record, hasRecord := resolveModelCatalogRecord(records, entry.ID)
	persisted, ok, _ := billingPricingSnapshotModel(pricingSnapshot, entry.ID)
	if !ok && hasRecord && record != nil {
		persisted = billingPricingPersistedModelFromRecord(record, rules)
		ok = true
	}
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

	provider := NormalizeModelProvider(firstRegistryString(entry.Provider, persisted.Provider))
	modelID := NormalizeModelCatalogModelID(firstRegistryString(persisted.Model, entry.ID))
	displayName := strings.TrimSpace(firstRegistryString(entry.DisplayName, persisted.DisplayName))
	if displayName == "" {
		displayName = FormatModelCatalogDisplayName(modelID)
	}

	mode := strings.TrimSpace(strings.ToLower(firstRegistryString(persisted.Mode)))
	if mode == "" {
		mode = inferModelMode(modelID, "")
	}

	return PublicModelCatalogItem{
		EntryID:           publicModelCatalogEntryID(0, "", modelID),
		PublicModelID:     modelID,
		Model:             modelID,
		BaseModel:         modelID,
		SourceModelID:     modelID,
		DisplayName:       displayName,
		Provider:          provider,
		ProviderIconKey:   provider,
		Status:            PublicModelStatusInfo,
		AvailabilityState: AccountModelAvailabilityUnknown,
		StaleState:        AccountModelStaleStateUnverified,
		LifecycleStatus: normalizePublicModelLifecycleStatus(
			entry.Status,
			entry.DisplayName,
			entry.ID,
		),
		RequestProtocols:     publicModelCatalogRequestProtocols(entry, provider),
		Mode:                 mode,
		Currency:             defaultModelPricingCurrency(persisted.Currency),
		PriceDisplay:         priceDisplay,
		OfficialPriceDisplay: officialDisplay,
		SalePriceDisplay:     priceDisplay,
		MultiplierSummary:    publicModelCatalogMultiplierSummaryFromForm(persisted.SaleForm),
		RuntimePriceSpec:     publicModelCatalogRuntimePriceSpecFromPersisted(persisted),
	}, true
}

func publicModelCatalogRuntimePriceSpecFromPersisted(persisted BillingPricingPersistedModel) PublicModelCatalogRuntimePriceSpec {
	return PublicModelCatalogRuntimePriceSpec{
		Currency:                        defaultModelPricingCurrency(persisted.Currency),
		OutputChargeSlot:                billingDefaultOutputChargeSlot(defaultString(persisted.OutputChargeSlot, persisted.Mode)),
		LongContextInputTokenThreshold:  persisted.LongContextInputTokenThreshold,
		LongContextInputCostMultiplier:  persisted.LongContextInputCostMultiplier,
		LongContextOutputCostMultiplier: persisted.LongContextOutputCostMultiplier,
	}
}

func publicModelCatalogPriceDisplayFromForm(
	metadata billingPricingFormMetadata,
	form BillingPricingLayerForm,
) PublicModelCatalogPriceDisplay {
	form = normalizeBillingPricingLayerFormForLayer(form, BillingLayerSale)
	primaryIDs := publicModelCatalogPrimaryFieldIDs(metadata)
	secondaryIDs := []string{
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

	display := PublicModelCatalogPriceDisplay{
		Primary:   make([]PublicModelCatalogPriceEntry, 0, len(primaryIDs)),
		Secondary: make([]PublicModelCatalogPriceEntry, 0, len(secondaryIDs)),
	}
	for _, fieldID := range primaryIDs {
		if entry, ok := publicModelCatalogPriceEntryForField(metadata, form, fieldID); ok {
			display.Primary = append(display.Primary, entry)
		}
	}
	for _, fieldID := range secondaryIDs {
		if containsString(primaryIDs, fieldID) {
			continue
		}
		if entry, ok := publicModelCatalogPriceEntryForField(metadata, form, fieldID); ok {
			display.Secondary = append(display.Secondary, entry)
		}
	}
	if len(display.Secondary) == 0 {
		display.Secondary = nil
	}
	return display
}

func publicModelCatalogPrimaryFieldIDs(metadata billingPricingFormMetadata) []string {
	if metadata.OutputChargeSlot == BillingChargeSlotTextOutput {
		if metadata.InputSupported {
			return []string{
				billingDiscountFieldInputPrice,
				billingDiscountFieldOutputPrice,
				billingDiscountFieldCachePrice,
				billingDiscountFieldBatchCachePrice,
			}
		}
		return []string{
			billingDiscountFieldOutputPrice,
			billingDiscountFieldCachePrice,
			billingDiscountFieldBatchCachePrice,
		}
	}
	return []string{billingDiscountFieldOutputPrice}
}

func publicModelCatalogPriceEntryForField(
	metadata billingPricingFormMetadata,
	form BillingPricingLayerForm,
	fieldID string,
) (PublicModelCatalogPriceEntry, bool) {
	value := billingPricingEffectiveFieldValue(form, fieldID)
	if value == nil {
		return PublicModelCatalogPriceEntry{}, false
	}
	return PublicModelCatalogPriceEntry{
		ID:    fieldID,
		Unit:  publicModelCatalogFieldUnit(metadata, fieldID),
		Value: *value,
	}, true
}

func publicModelCatalogFieldUnit(metadata billingPricingFormMetadata, fieldID string) string {
	switch fieldID {
	case billingDiscountFieldInputPrice, billingDiscountFieldInputPriceAboveThreshold, billingDiscountFieldBatchInputPrice:
		return billingUnitForChargeSlot(BillingChargeSlotTextInput)
	case billingDiscountFieldOutputPrice, billingDiscountFieldOutputPriceAboveThreshold, billingDiscountFieldBatchOutputPrice:
		return billingUnitForChargeSlot(metadata.OutputChargeSlot)
	case billingDiscountFieldCachePrice, billingDiscountFieldBatchCachePrice:
		return billingUnitForChargeSlot(BillingChargeSlotCacheCreate)
	case billingDiscountFieldGroundingSearch:
		return billingUnitForChargeSlot(BillingChargeSlotGroundingSearchRequest)
	case billingDiscountFieldGroundingMaps:
		return billingUnitForChargeSlot(BillingChargeSlotGroundingMapsRequest)
	case billingDiscountFieldFileSearchEmbedding:
		return billingUnitForChargeSlot(BillingChargeSlotFileSearchEmbeddingToken)
	case billingDiscountFieldFileSearchRetrieval:
		return billingUnitForChargeSlot(BillingChargeSlotFileSearchRetrievalToken)
	default:
		return ""
	}
}

func publicModelCatalogMultiplierSummaryFromForm(form BillingPricingLayerForm) PublicModelCatalogMultiplierSummary {
	values := billingPricingConfiguredMultiplierValues(form)
	if len(values) == 0 {
		return PublicModelCatalogMultiplierSummary{
			Enabled: false,
			Kind:    publicModelCatalogMultiplierDisabled,
		}
	}

	var (
		firstValue float64
		hasFirst   bool
		mixed      bool
	)
	for _, value := range values {
		if !hasFirst {
			firstValue = value
			hasFirst = true
			continue
		}
		if !billingPricesAlmostEqual(firstValue, value) {
			mixed = true
			break
		}
	}
	if mixed {
		return PublicModelCatalogMultiplierSummary{
			Enabled: true,
			Kind:    publicModelCatalogMultiplierMixed,
			Mode:    string(normalizeBillingPricingMultiplierMode(form.MultiplierMode)),
		}
	}
	return PublicModelCatalogMultiplierSummary{
		Enabled: true,
		Kind:    publicModelCatalogMultiplierUniform,
		Mode:    string(normalizeBillingPricingMultiplierMode(form.MultiplierMode)),
		Value:   modelCatalogFloat64Ptr(firstValue),
	}
}

func publicModelCatalogRequestProtocols(entry modelregistry.ModelEntry, provider string) []string {
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
	for _, value := range entry.Platforms {
		appendProtocol(value)
	}
	for _, value := range entry.ProtocolIDs {
		appendProtocol(value)
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

func publicModelCatalogProtocolFamily(value string) string {
	normalized := strings.TrimSpace(strings.ToLower(value))
	switch {
	case normalized == "" || normalized == "*":
		return ""
	case normalized == PlatformOpenAI || strings.HasPrefix(normalized, "gpt") || strings.HasPrefix(normalized, "codex") || strings.HasPrefix(normalized, "o1") || strings.HasPrefix(normalized, "o3") || strings.HasPrefix(normalized, "o4"):
		return PlatformOpenAI
	case normalized == PlatformAnthropic || normalized == PlatformKiro || strings.HasPrefix(normalized, "claude"):
		return PlatformAnthropic
	case normalized == PlatformGemini || strings.HasPrefix(normalized, "gemini"):
		return PlatformGemini
	case normalized == PlatformGrok || strings.HasPrefix(normalized, "grok"):
		return PlatformGrok
	case normalized == PlatformAntigravity || strings.Contains(normalized, "antigravity"):
		return PlatformAntigravity
	case normalized == "vertex" || strings.Contains(normalized, "vertex") || strings.Contains(normalized, "batch"):
		return publicModelCatalogProtocolVertex
	default:
		return ""
	}
}

func computePublicModelCatalogETagWithPageSize(pageSize int, items []PublicModelCatalogItem) (string, error) {
	payload, err := json.Marshal(struct {
		PageSize int                      `json:"page_size,omitempty"`
		Items    []PublicModelCatalogItem `json:"items"`
	}{
		PageSize: pageSize,
		Items:    items,
	})
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(payload)
	return "W/\"" + hex.EncodeToString(sum[:]) + "\"", nil
}

func clonePublicModelCatalogSnapshot(snapshot *PublicModelCatalogSnapshot) *PublicModelCatalogSnapshot {
	if snapshot == nil {
		return nil
	}
	cloned := &PublicModelCatalogSnapshot{
		ETag:          snapshot.ETag,
		UpdatedAt:     snapshot.UpdatedAt,
		PageSize:      snapshot.PageSize,
		CatalogSource: snapshot.CatalogSource,
		Items:         make([]PublicModelCatalogItem, 0, len(snapshot.Items)),
	}
	for _, item := range snapshot.Items {
		cloned.Items = append(cloned.Items, clonePublicModelCatalogItem(item))
	}
	return cloned
}

func clonePublicModelCatalogItem(item PublicModelCatalogItem) PublicModelCatalogItem {
	cloned := item
	cloned.RequestProtocols = append([]string(nil), item.RequestProtocols...)
	cloned.SourceIDs = append([]string(nil), item.SourceIDs...)
	cloned.PriceDisplay = PublicModelCatalogPriceDisplay{
		Primary:   clonePublicModelCatalogPriceEntries(item.PriceDisplay.Primary),
		Secondary: clonePublicModelCatalogPriceEntries(item.PriceDisplay.Secondary),
	}
	cloned.OfficialPriceDisplay = PublicModelCatalogPriceDisplay{
		Primary:   clonePublicModelCatalogPriceEntries(item.OfficialPriceDisplay.Primary),
		Secondary: clonePublicModelCatalogPriceEntries(item.OfficialPriceDisplay.Secondary),
	}
	cloned.SalePriceDisplay = PublicModelCatalogPriceDisplay{
		Primary:   clonePublicModelCatalogPriceEntries(item.SalePriceDisplay.Primary),
		Secondary: clonePublicModelCatalogPriceEntries(item.SalePriceDisplay.Secondary),
	}
	cloned.MultiplierSummary = PublicModelCatalogMultiplierSummary{
		Enabled: item.MultiplierSummary.Enabled,
		Kind:    item.MultiplierSummary.Kind,
		Mode:    item.MultiplierSummary.Mode,
		Value:   modelCatalogFloat64PtrValue(item.MultiplierSummary.Value),
	}
	cloned.RuntimePriceSpec = normalizePublicModelCatalogRuntimePriceSpec(item.RuntimePriceSpec)
	return cloned
}

func clonePublicModelCatalogItems(items []PublicModelCatalogItem) []PublicModelCatalogItem {
	if len(items) == 0 {
		return nil
	}
	cloned := make([]PublicModelCatalogItem, 0, len(items))
	for _, item := range items {
		cloned = append(cloned, clonePublicModelCatalogItem(item))
	}
	return cloned
}

func sanitizePublicModelCatalogSnapshotForPublic(snapshot *PublicModelCatalogSnapshot) *PublicModelCatalogSnapshot {
	cloned := clonePublicModelCatalogSnapshot(snapshot)
	if cloned == nil {
		return nil
	}
	cloned.Items = sanitizePublicModelCatalogItemsForPublic(cloned.Items)
	return cloned
}

func sanitizePublicModelCatalogItemsForPublic(items []PublicModelCatalogItem) []PublicModelCatalogItem {
	if len(items) == 0 {
		return nil
	}
	sanitized := make([]PublicModelCatalogItem, 0, len(items))
	for _, item := range items {
		sanitized = append(sanitized, sanitizePublicModelCatalogItemForPublic(item))
	}
	return sanitized
}

func sanitizePublicModelCatalogItemForPublic(item PublicModelCatalogItem) PublicModelCatalogItem {
	cloned := clonePublicModelCatalogItem(item)
	cloned.SourceAccountID = 0
	cloned.SourceAccountName = ""
	cloned.RuntimePriceSpec = PublicModelCatalogRuntimePriceSpec{}
	if cloned.PublicModelID == "" {
		cloned.PublicModelID = cloned.Model
	}
	if cloned.Model == "" {
		cloned.Model = cloned.PublicModelID
	}
	return cloned
}

func normalizePublicModelCatalogRuntimePriceSpec(spec PublicModelCatalogRuntimePriceSpec) PublicModelCatalogRuntimePriceSpec {
	spec.Currency = defaultModelPricingCurrency(spec.Currency)
	spec.OutputChargeSlot = billingDefaultOutputChargeSlot(spec.OutputChargeSlot)
	if spec.LongContextInputTokenThreshold < 0 {
		spec.LongContextInputTokenThreshold = 0
	}
	if spec.LongContextInputCostMultiplier < 0 {
		spec.LongContextInputCostMultiplier = 0
	}
	if spec.LongContextOutputCostMultiplier < 0 {
		spec.LongContextOutputCostMultiplier = 0
	}
	return spec
}

func clonePublicModelCatalogPriceEntries(entries []PublicModelCatalogPriceEntry) []PublicModelCatalogPriceEntry {
	if len(entries) == 0 {
		return nil
	}
	cloned := make([]PublicModelCatalogPriceEntry, len(entries))
	copy(cloned, entries)
	return cloned
}

func modelCatalogFloat64PtrValue(value *float64) *float64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
